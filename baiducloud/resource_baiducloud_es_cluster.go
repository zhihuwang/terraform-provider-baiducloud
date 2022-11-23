/*
Use this resource to create a es cluster.

Example Usage

```hcl
resource "baiducloud_es_cluster" "default_managed" {

}
```
*/
package baiducloud

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/baidubce/bce-sdk-go/services/bes"
	estypes "github.com/baidubce/bce-sdk-go/services/cce/v2/types"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/terraform-providers/terraform-provider-baiducloud/baiducloud/connectivity"
)

func resourceBaiduCloudESCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceBaiduCloudESClusterCreate,
		Update: resourceBaiduCloudESClusterUpdate,
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
			"kibana_url": {
				Type:        schema.TypeString,
				Description: "kibana url",
				Computed:    true,
			},
			"kibana_eip": {
				Type:        schema.TypeString,
				Description: "kibana eip",
				Computed:    true,
			},
			"es_url": {
				Type:        schema.TypeString,
				Description: "es url",
				Computed:    true,
			},
			"es_eip": {
				Type:        schema.TypeString,
				Description: "es eip",
				Computed:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "name",
				Required:    true,
			},
			"username": {
				Type:        schema.TypeString,
				Description: "username",
				Optional:    true,
			},
			"password": {
				Type:        schema.TypeString,
				Description: "password",
				Optional:    true,
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
				Description: "Payment Type",
				Optional:    true,
				Default:     "postpay",
			},
			"payment_time": {
				Type:        schema.TypeInt,
				Description: "Payment Time",
				Optional:    true,
				Default:     12,
			},
			"auto_renew": {
				Type:        schema.TypeBool,
				Description: "auto renew",
				Optional:    true,
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
		if !response.Success {
			return resource.NonRetryableError(WrapErrorf(fmt.Errorf("[Code: %s; Message: %s; RequestId: %s]", response.Code, response.Error.Message, response.Error.RequestId), DefaultErrorMsg, "baiducloud_es_cluster", action, BCESDKGoERROR))
		}
		d.SetId(response.Result.ClusterId)
		return nil
	})
	if err != nil {
		log.Printf("Create Cluster Error:" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_es_cluster", action, BCESDKGoERROR)
	}

	stateConf := buildStateConf(
		[]string{string("Creating"), string("Initializing"), string("Starting")},
		[]string{string("Avaiable"), string("Running")},
		d.Timeout(schema.TimeoutCreate),
		besService.ClusterStateRefreshes(d.Id(), []string{
			"Failed",
		}),
	)
	if _, err := stateConf.WaitForState(); err != nil {
		log.Printf("Create Cluster Error:" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_es_cluster", action, BCESDKGoERROR)
	}

	return resourceBaiduCloudESClusterRead(d, meta)
}
func resourceBaiduCloudESClusterUpdate(d *schema.ResourceData, meta interface{}) error {

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

	d.Set("cluster_status", response.Result.ActualStatus)
	d.Set("kibana_url", response.Result.KibanaURL)
	d.Set("kibana_eip", response.Result.KibanaEip)
	d.Set("es_url", response.Result.EsURL)
	d.Set("es_eip", response.Result.EsEip)
	d.Set("username", response.Result.AdminUsername)
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
			"Deleting",
			"Stopped",
			"Audit_stopping",
			"Audit_stopped",
		},
		[]string{"Deleted", "OrderFailed"},
		d.Timeout(schema.TimeoutDelete),
		besService.ClusterStateRefreshes(clusterId, []string{
			"Failed",
		}),
	)
	if _, err := stateConf.WaitForState(); err != nil {
		log.Printf("Delete Cluster Error:" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_es_cluster", action, BCESDKGoERROR)
	}
	time.Sleep(10 * time.Second) //waiting for infrastructure delete before delete vpc & security group
	return nil
}
