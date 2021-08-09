/*
Use this resource to create a CCEv2 InstanceGroup.

~> **NOTE:** The create/update/delete operation of ccev2 does NOT take effect immediately，maybe takes for several minutes.

Example Usage

```hcl
resource "baiducloud_ccev2_instance_group_replica" "ccev2_instance_group_replica_1" {
    cluster_id = baiducloud_ccev2_cluster.default_custom.id
    replicas_change = 1
}
```
*/
package baiducloud

import (
	"errors"
	"log"
	"time"

	ccev2 "github.com/baidubce/bce-sdk-go/services/cce/v2"
	"github.com/baidubce/bce-sdk-go/services/cce/v2/types"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-baiducloud/baiducloud/connectivity"
)

func resourceBaiduCloudCCEv2InstanceGroupReplica() *schema.Resource {
	return &schema.Resource{
		Create: resourceBaiduCloudCCEv2InstanceGroupReplicaCreate,
		Read:   resourceBaiduCloudCCEv2InstanceGroupReplicaRead,
		Delete: resourceBaiduCloudCCEv2InstanceGroupReplicaDelete,
		Update: resourceBaiduCloudCCEv2InstanceGroupReplicaUpdate,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
			Update: schema.DefaultTimeout(20 * time.Minute),
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:        schema.TypeString,
				Description: "Cluster ID of Instance Group",
				ForceNew:    true,
				Required:    true,
			},
			"replicas_change": {
				Type:        schema.TypeInt,
				Description: "Number of instances in this Instance Group",
				Optional:    true,
				Default:     0,
			},
			"total": {
				Type:        schema.TypeInt,
				Description: "Number of instances in the cluster",
				Optional:    true,
				Computed:    true,
			},
			"instance_group_id": {
				Type:        schema.TypeString,
				Description: "id of Instance Group",
				Optional:    true,
				Computed:    true,
			},
			"replicas": {
				Type:        schema.TypeInt,
				Description: "Number of instances in this Instance Group",
				Optional:    true,
				Computed:    true,
			},
			"ready_replicas": {
				Type:        schema.TypeInt,
				Description: "Number of instances in RUNNING of this Instance Group",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}
func getInstanceGroup(cluster_id string, left_nodes int, client *connectivity.BaiduClient) ([]*ccev2.InstanceGroup, error) {
	args := &ccev2.ListInstanceGroupsArgs{}
	args.ClusterID = cluster_id
	listOpts := &ccev2.InstanceGroupListOption{}
	listOpts.PageSize = 200
	args.ListOption = listOpts

	action := "Get CCEv2 InstanceGroups by Cluster ID:" + args.ClusterID
	raw, err := client.WithCCEv2Client(func(client *ccev2.Client) (i interface{}, e error) {
		return client.ListInstanceGroups(args)
	})
	if err != nil {
		log.Printf("List InstanceGroup Instances Error:" + err.Error())
		return nil, WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_instance_group_repica", action, BCESDKGoERROR)
	}
	addDebug(action, raw)
	response := raw.(*ccev2.ListInstanceGroupResponse)
	if response.Page.List == nil {
		err := errors.New("instance list is nil")
		log.Printf("List InstanceGroup Instances Error:" + err.Error())
		return nil, WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_instance_groups", action, BCESDKGoERROR)
	}
	groups := getAvailableInstanceGroup(response.Page.List, types.ClusterRoleNode, left_nodes)
	return groups, nil
}
func setRepica(d *schema.ResourceData, meta interface{}, client *connectivity.BaiduClient) error {
	clusterId := d.Id()
	if change, ok := d.GetOk("replicas_change"); ok {
		if change.(int) != 0 {
			args := &ccev2.UpdateInstanceGroupReplicasArgs{
				ClusterID: clusterId,
				Request: &ccev2.UpdateInstanceGroupReplicasRequest{
					DeleteInstance: true,
					DeleteOption: &types.DeleteOption{
						DeleteResource:    true,
						DeleteCDSSnapshot: true,
					},
				},
			}
			action := "Update CCEv2 Cluster Instance Group Repica "
			groups, err := getInstanceGroup(clusterId, change.(int), client)
			if err != nil {
				return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_cluster_replica", action, BCESDKGoERROR)
			}
			if len(groups) == 0 {
				return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_cluster_replica", action, "Instance Group is empty or can not be used!")
			}
			args.InstanceGroupID = groups[0].Spec.CCEInstanceGroupID
			args.Request.Replicas = groups[0].Spec.Replicas + change.(int)

			_, err = client.WithCCEv2Client(func(client *ccev2.Client) (interface{}, error) {
				return client.UpdateInstanceGroupReplicas(args)
			})
			if err != nil {
				return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_cluster_replica", action, BCESDKGoERROR)
			}
			//waiting all instance in instance group are ready
			createTimeOutTime := d.Timeout(schema.TimeoutCreate)
			loopsCount := createTimeOutTime.Microseconds() / ((10 * time.Second).Microseconds())
			var i int64
			for i = 1; i <= loopsCount; i++ {
				time.Sleep(5 * time.Second)
				argsGetInstanceGroup := &ccev2.GetInstanceGroupArgs{
					ClusterID:       args.ClusterID,
					InstanceGroupID: args.InstanceGroupID,
				}
				rawInstanceGroupResp, err := client.WithCCEv2Client(func(client *ccev2.Client) (interface{}, error) {
					return client.GetInstanceGroup(argsGetInstanceGroup)
				})
				if err != nil {
					return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_cluster_replica", action, BCESDKGoERROR)
				}
				instanceGroupResp := rawInstanceGroupResp.(*ccev2.GetInstanceGroupResponse)
				if instanceGroupResp.InstanceGroup.Status.ReadyReplicas == instanceGroupResp.InstanceGroup.Spec.Replicas {
					break
				}
				if i == loopsCount {
					return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_cluster_replica", action, BCESDKGoERROR)
				}
			}
			d.Set("instance_group_id", args.InstanceGroupID)
			d.Set("replicas_change", 0)
		}
	}
	return nil
}
func resourceBaiduCloudCCEv2InstanceGroupReplicaCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.BaiduClient)
	clusterId := d.Get("cluster_id")
	setRepica(d, meta, client)
	d.SetId(clusterId.(string))
	return resourceBaiduCloudCCEv2InstanceGroupReplicaRead(d, meta)
}

func resourceBaiduCloudCCEv2InstanceGroupReplicaRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.BaiduClient)
	clusterId := d.Id()
	action := "Get CCEv2 Cluster " + clusterId

	//1.Get Status of the Cluster
	raw, err := client.WithCCEv2Client(func(client *ccev2.Client) (interface{}, error) {
		return client.GetCluster(clusterId)
	})
	if err != nil {
		if NotFoundError(err) {
			log.Printf("Cluster Not Found. Set Resource ID to Empty.")
			d.SetId("") //Resource Not Found, make the ID of resource to empty to delete it in state file.
			return nil
		}
		log.Printf("Get Cluster Error:" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_cluster_replica", action, BCESDKGoERROR)
	}
	response := raw.(*ccev2.GetClusterResponse)
	if response == nil {
		err := Error("Response is nil")
		log.Printf("Get Cluster Error:" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_cluster_replica", action, BCESDKGoERROR)
	}
	d.Set("total", response.Cluster.Status.NodeNum)
	if groupId, ok := d.GetOk("instance_group_id"); ok {
		argsGetInstanceGroup := &ccev2.GetInstanceGroupArgs{
			ClusterID:       clusterId,
			InstanceGroupID: groupId.(string),
		}
		rawInstanceGroupResp, err := client.WithCCEv2Client(func(client *ccev2.Client) (interface{}, error) {
			return client.GetInstanceGroup(argsGetInstanceGroup)
		})
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_cluster_replica", action, BCESDKGoERROR)
		}
		instanceGroupResp := rawInstanceGroupResp.(*ccev2.GetInstanceGroupResponse)
		d.Set("ready_replicas", instanceGroupResp.InstanceGroup.Status.ReadyReplicas)
		d.Set("replicas", instanceGroupResp.InstanceGroup.Spec.Replicas)
	}

	return nil
}

func resourceBaiduCloudCCEv2InstanceGroupReplicaUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.BaiduClient)
	setRepica(d, meta, client)
	return resourceBaiduCloudCCEv2InstanceGroupReplicaRead(d, meta)
}

func resourceBaiduCloudCCEv2InstanceGroupReplicaDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
