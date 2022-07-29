/*
Use this resource to create a es cluster.

Example Usage

```hcl
resource "baiducloud_es_cluster" "default_managed" {
  cluster_spec  {
    cluster_name = var.cluster_name
    cluster_type = "normal"
    k8s_version = "1.16.8"
    runtime_type = "docker"
    vpc_id = baiducloud_vpc.default.id
    plugins = ["core-dns", "kube-proxy"]
    master_config {
      master_type = "managed"
      cluster_ha = 2
      exposed_public = false
      cluster_blb_vpc_subnet_id = baiducloud_subnet.defaultA.id
      managed_cluster_master_option {
        master_vpc_subnet_zone = "zoneA"
      }
    }
    container_network_config  {
      mode = "kubenet"
      lb_service_vpc_subnet_id = baiducloud_subnet.defaultA.id
      node_port_range_min = 30000
      node_port_range_max = 32767
      max_pods_per_node = 64
      cluster_pod_cidr = var.cluster_pod_cidr
      cluster_ip_service_cidr = var.cluster_ip_service_cidr
      ip_version = "ipv4"
      kube_proxy_mode = "iptables"
    }
    cluster_delete_option {
      delete_resource = true
      delete_cds_snapshot = true
    }
  }
}
```
*/
package baiducloud

import (
	"errors"
	"log"
	"time"

	"github.com/baidubce/bce-sdk-go/services/bes"
	estypes "github.com/baidubce/bce-sdk-go/services/cce/v2/types"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-baiducloud/baiducloud/connectivity"
)

func resourceBaiduCloudESCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceBaiduCloudESClusterCreate,
		Read:   resourceBaiduCloudESClusterRead,
		Delete: resourceBaiduCloudESClusterDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
			Update: schema.DefaultTimeout(20 * time.Minute),
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			//Params for creating the cluster
			"modules": {
				Type:        schema.TypeList,
				Description: "Specification of the cluster",
				Required:    true,
				ForceNew:    true,
				MaxItems:    5,
				Elem:        resourceBESClusterModule(),
			},
			//Status of the cluster
			"cluster_status": {
				Type:        schema.TypeString,
				Description: "status of the cluster",
				Computed:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "name",
				Required:    true,
			},
			"password": {
				Type:        schema.TypeString,
				Description: "password",
				Required:    true,
			},
			"security_group_id": {
				Type:        schema.TypeString,
				Description: "security_group_id",
				Required:    true,
			},
			"subnet_id": {
				Type:        schema.TypeString,
				Description: "subnet Uuid",
				Required:    true,
			},
			"available_zone": {
				Type:        schema.TypeString,
				Description: "availableZone",
				Required:    true,
			},
			"vpc_id": {
				Type:        schema.TypeString,
				Description: "vpcId",
				Required:    true,
			},
			"version": {
				Type:        schema.TypeString,
				Description: "version",
				Optional:    true,
				Default:     "7.4.2",
			},
			"payment_type": {
				Type:        schema.TypeString,
				Description: "paymentType",
				Optional:    true,
				Default:     "postpay",
			},
			"payment_time": {
				Type:        schema.TypeInt,
				Description: "payment_time",
				Optional:    true,
				Default:     0,
			},
		},
	}
}

func resourceBaiduCloudESClusterCreate(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*connectivity.BaiduClient)
	besService := BesService{client}

	createClusterArgs, err := buildBESCreateClusterRequest(d)
	if err != nil {
		log.Printf("Build CreateClusterArgs Error:" + err.Error())
		return WrapError(err)
	}

	action := "Create es cluster " + createClusterArgs.Name
	err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		raw, err := client.WithBesClient(func(client *bes.Client) (interface{}, error) {
			return client.CreateCluster(createClusterArgs)
		})
		if err != nil {
			return resource.NonRetryableError(err)
		}
		addDebug(action, raw)
		response, ok := raw.(*bes.ESClusterResponse)
		if !ok {
			err = errors.New("response format illegal")
			return resource.NonRetryableError(err)
		}
		d.SetId(response.Result.ClusterId)
		return nil
	})
	if err != nil {
		log.Printf("Create Cluster Error:" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_es_cluster", action, BCESDKGoERROR)
	}

	stateConf := buildStateConf(
		[]string{string("Creating"), string("Starting")},
		[]string{string("Avaiable"), string("Running")},
		d.Timeout(schema.TimeoutCreate),
		besService.ClusterStateRefreshes(d.Id(), []string{
			"Running",
			"Falied",
		}),
	)
	if _, err := stateConf.WaitForState(); err != nil {
		log.Printf("Create Cluster Error:" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_es_cluster", action, BCESDKGoERROR)
	}

	return resourceBaiduCloudESClusterRead(d, meta)
}

func resourceBaiduCloudESClusterRead(d *schema.ResourceData, meta interface{}) error {
	clusterId := d.Id()
	action := "Get es Cluster " + clusterId
	client := meta.(*connectivity.BaiduClient)

	//1.Get Status of the Cluster
	raw, err := client.WithBesClient(func(client *bes.Client) (interface{}, error) {
		return client.GetCluster(&bes.GetESClusterRequest{
			ClusterId: clusterId,
		})
	})
	if err != nil {
		if NotFoundError(err) {
			log.Printf("Cluster Not Found. Set Resource ID to Empty.")
			d.SetId("") //Resource Not Found, make the ID of resource to empty to delete it in state file.
			return nil
		}
		log.Printf("Get Cluster Error:" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_es_cluster", action, BCESDKGoERROR)
	}
	response := raw.(*bes.DetailESClusterResponse)
	if response == nil {
		err := Error("Response is nil")
		log.Printf("Get Cluster Error:" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_es_cluster", action, BCESDKGoERROR)
	}

	err = d.Set("cluster_status", response.Result.ActualStatus)
	if err != nil {
		log.Printf("Set cluster_status Error:" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_es_cluster", action, BCESDKGoERROR)
	}

	return nil
}

func resourceBaiduCloudESClusterDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.BaiduClient)
	clusterId := d.Id()
	besService := BesService{client}
	action := "Delete es Cluster " + clusterId
	err := resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		raw, err := client.WithBesClient(func(client *bes.Client) (interface{}, error) {
			return client.DeleteCluster(&bes.GetESClusterRequest{
				ClusterId: clusterId,
			})
		})
		if err != nil {
			return resource.NonRetryableError(err)
		}
		addDebug(action, raw)
		return nil
	})
	if err != nil {
		log.Printf("Delete Cluster Error:" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_es_cluster", action, BCESDKGoERROR)
	}

	stateConf := buildStateConf(
		[]string{string("Running"), string(estypes.ClusterPhaseRunning),
			string(estypes.ClusterPhaseDeleting),
			string(estypes.ClusterPhaseCreateFailed),
			string(estypes.ClusterPhaseProvisioned),
			string(estypes.ClusterPhaseProvisioning),
			string(estypes.ClusterPhaseDeleteFailed),
		},
		[]string{string(estypes.ClusterPhaseDeleted)},
		d.Timeout(schema.TimeoutDelete),
		besService.ClusterStateRefreshes(clusterId, []string{
			"failed",
		}),
	)
	if _, err := stateConf.WaitForState(); err != nil {
		log.Printf("Delete Cluster Error:" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_es_cluster", action, BCESDKGoERROR)
	}
	time.Sleep(1 * time.Minute) //waiting for infrastructure delete before delete vpc & security group
	return nil
}
