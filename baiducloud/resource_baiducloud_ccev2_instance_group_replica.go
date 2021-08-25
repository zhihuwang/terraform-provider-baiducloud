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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"text/template"
	"time"

	"github.com/baidubce/bce-sdk-go/services/cce"
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
				Required:    true,
			},
			"tpl": {
				Type:        schema.TypeString,
				Description: "the params template for baidu cce v1 scale up",
				Optional:    true,
			},
			"vars": {
				Type:        schema.TypeMap,
				Description: "variables used in params template",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
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

func setRepicaV1(d *schema.ResourceData, meta interface{}, client *connectivity.BaiduClient) error {
	action := "scale up  CCE v1 Cluster"
	clusterId := d.Get("cluster_id").(string)
	bccService := BccService{client}
	cceService := CceService{client}
	if !strings.HasPrefix(clusterId, "c-") {
		return nil
	}
	if change, ok := d.GetOk("replicas_change"); ok {
		if change.(int) <= 0 {
			return WrapErrorf(errors.New("replicas_change should be great than 0"), DefaultErrorMsg, "baiducloud_ccev1_cluster_replica", action, "[Parameters Error]")
		}
		cceResponse, err := cceService.GetCceV1Cluster(clusterId, client)
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev1_cluster_replica", action, BCESDKGoERROR)
		}
		slaveVmCount := cceResponse.SlaveVmCount
		log.Printf("SlaveVmCount=%d", slaveVmCount)
		tpl, tplok := d.GetOk("tpl")
		vars, varok := d.GetOk("vars")
		if !tplok || !varok {
			return WrapErrorf(errors.New("tpl and vars is required for baidu cce v1"), DefaultErrorMsg, "baiducloud_ccev1_cluster_replica", action, BCESDKGoERROR)
		}
		tplstr := tpl.(string)
		textTpl, err := template.New("params").Delims("[[", "]]").Parse(tplstr)
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev1_cluster_replica", action, "[Template Parse error]")
		}
		var tmplBytes bytes.Buffer
		data := map[string]interface{}{}
		data["vars"] = vars
		err = textTpl.Execute(&tmplBytes, data)
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev1_cluster_replica", action, "[Template Execute error]")
		}
		log.Printf("BccConfig=%s", tmplBytes.String())
		var args cce.ScalingUpArgs
		err = json.Unmarshal(tmplBytes.Bytes(), &args)
		jsonStr, err1 := json.Marshal(&args)
		log.Printf("ScalingUpArgs=%s, err=%v", string(jsonStr), err1)
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev1_cluster_replica", action, "[Unmarshal ScalingUpArgs failed]"+tmplBytes.String())
		}
		raw, err := client.WithCCEClient(func(client *cce.Client) (i interface{}, e error) {
			return client.ScalingUp(&args)
		})
		addDebug(action, raw)

		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev1_cluster_replica", action, BCESDKGoERROR)
		}

		log.Printf("ScalingUpResult=%v", raw)
		//waiting all instance in instance group are ready
		createTimeOutTime := d.Timeout(schema.TimeoutCreate)
		loopsCount := createTimeOutTime.Microseconds() / ((10 * time.Second).Microseconds())
		var i int64
		newNodeSize := change.(int)
		for i = 1; i <= loopsCount; i++ {
			time.Sleep(5 * time.Second)
			cceResponseRefresh, err := cceService.GetCceV1Cluster(clusterId, client)
			if err != nil {
				return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev1_cluster_replica", action, BCESDKGoERROR)
			}
			currentSlaveVmCount := cceResponseRefresh.SlaveVmCount
			log.Printf("currentSlaveVmCount=%d", currentSlaveVmCount)
			log.Printf("targetSlaveVmCount=%d", slaveVmCount+change.(int))
			if currentSlaveVmCount == slaveVmCount+newNodeSize {
				listNodeResult, err := cceService.GetCceV1Nodes(clusterId, client)
				if err == nil {
					var m int
					var failedNodes []string
					var finishedNodes []string
					for m = 0; m < newNodeSize; m++ {
						if listNodeResult.Nodes[m].Status == "CREATE_FAILED" {
							failedNodes = append(failedNodes, listNodeResult.Nodes[m].InstanceShortId)
						} else if listNodeResult.Nodes[m].Status == "RUNNING" {
							finishedNodes = append(finishedNodes, listNodeResult.Nodes[m].InstanceShortId)
						}
					}
					if len(failedNodes) > 0 {
						return fmt.Errorf("CCE v1 Node CREATE_FAILED:%v", failedNodes)
					}
					finished := 0
					if len(finishedNodes) == newNodeSize {
						for m = 0; m < newNodeSize; m++ {
							err := bccService.EnablePrepaidAndAutoRenew(listNodeResult.Nodes[m].InstanceShortId)
							if err == nil {
								finished++
							} else {
								log.Printf("EnablePrepaidAndAutoRenew for bcc instance[%s] failed:%s", listNodeResult.Nodes[m].InstanceShortId, err.Error())
							}
						}
						if finished == newNodeSize {
							break
						}
					}
				} else {
					log.Printf("get cce v1 nodes failed:%s", err.Error())
				}

			}
			if i == loopsCount {
				return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_cluster_replica", action, BCESDKGoERROR)
			}
		}
		d.Set("replicas_change", 0)
	} else {
		return errors.New("replicas_change is invalid")
	}
	return nil

}

func setRepicaV2(d *schema.ResourceData, meta interface{}, client *connectivity.BaiduClient) error {
	clusterId := d.Get("cluster_id")
	ccev2Service := Ccev2Service{client}
	bccService := BccService{client}
	if !strings.HasPrefix(clusterId.(string), "cce-") {
		return nil
	}
	if change, ok := d.GetOk("replicas_change"); ok {
		if change.(int) != 0 {
			args := &ccev2.UpdateInstanceGroupReplicasArgs{
				ClusterID: clusterId.(string),
				Request: &ccev2.UpdateInstanceGroupReplicasRequest{
					DeleteInstance: true,
					DeleteOption: &types.DeleteOption{
						DeleteResource:    true,
						DeleteCDSSnapshot: true,
					},
				},
			}
			action := "Update CCEv2 Cluster Instance Group Repica "
			groups, err := ccev2Service.GetInstanceGroupList(clusterId.(string), change.(int))
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
			newNodeSize := change.(int)
			for i = 1; i <= loopsCount; i++ {
				time.Sleep(5 * time.Second)
				instanceGroupResp, err := ccev2Service.GetInstanceGroupDetail(args.ClusterID, args.InstanceGroupID)
				if err != nil {
					return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_cluster_replica", action, BCESDKGoERROR)
				}
				if instanceGroupResp.Status.ReadyReplicas == instanceGroupResp.Spec.Replicas {
					instancesResponse, err := ccev2Service.GetInstanceGroupInstances(&ccev2.ListInstanceByInstanceGroupIDArgs{
						ClusterID:       args.ClusterID,
						InstanceGroupID: args.InstanceGroupID,
					})
					if err != nil {
						return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_cluster_replica", action, BCESDKGoERROR)
					}
					if err == nil {
						finished := 0
						var m int
						for m = 0; m < newNodeSize; m++ {
							err := bccService.EnablePrepaidAndAutoRenew(instancesResponse.Page.List[m].Status.Machine.InstanceID)
							if err == nil {
								finished++
							} else {
								log.Printf("EnablePrepaidAndAutoRenew for bcc instance[%s] failed:%s", instancesResponse.Page.List[m].Status.Machine.InstanceID, err.Error())
							}
						}
						if finished == newNodeSize {
							break
						}
					} else {
						log.Printf("Get InstanceGroup Instances [%s]->[%s] failed:%s", args.ClusterID, args.InstanceGroupID, err.Error())
					}
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
	d.SetId(clusterId.(string))
	if !strings.HasPrefix(clusterId.(string), "c-") && !strings.HasPrefix(clusterId.(string), "cce-") {
		return errors.New("invalid cluster_id:" + clusterId.(string))
	}
	err1 := setRepicaV1(d, meta, client)
	if err1 != nil {
		return err1
	}
	err2 := setRepicaV2(d, meta, client)
	if err2 != nil {
		return err2
	}

	return resourceBaiduCloudCCEv2InstanceGroupReplicaRead(d, meta)
}
func readV2(clusterId string, d *schema.ResourceData, client *connectivity.BaiduClient) error {
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
func readV1(clusterId string, d *schema.ResourceData, client *connectivity.BaiduClient) error {
	action := "Get CCEv1 Cluster " + clusterId
	cceService := CceService{client}
	//1.Get Status of the Cluster
	result, err := cceService.GetCceV1Cluster(clusterId, client)
	if err != nil {
		if NotFoundError(err) {
			log.Printf("Cluster Not Found. Set Resource ID to Empty.")
			d.SetId("") //Resource Not Found, make the ID of resource to empty to delete it in state file.
			return nil
		}
		log.Printf("Get Cluster Error:" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev1_cluster_replica", action, BCESDKGoERROR)
	}
	if result == nil {
		err := Error("Response is nil")
		log.Printf("Get Cluster Error:" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev1_cluster_replica", action, BCESDKGoERROR)
	}
	d.Set("total", result.SlaveVmCount)
	d.Set("ready_replicas", result.SlaveVmCount)
	d.Set("replicas", result.SlaveVmCount)
	return nil
}
func resourceBaiduCloudCCEv2InstanceGroupReplicaRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.BaiduClient)
	clusterId := d.Id()
	if !strings.HasPrefix(clusterId, "cce-") {
		readV2(clusterId, d, client)
	} else {
		readV1(clusterId, d, client)
	}
	return nil
}

func resourceBaiduCloudCCEv2InstanceGroupReplicaUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.BaiduClient)
	clusterId := d.Get("cluster_id")
	if !strings.HasPrefix(clusterId.(string), "c-") && !strings.HasPrefix(clusterId.(string), "cce-") {
		return errors.New("invalid cluster_id:" + clusterId.(string))
	}
	err1 := setRepicaV1(d, meta, client)
	if err1 != nil {
		return err1
	}
	err2 := setRepicaV2(d, meta, client)
	if err2 != nil {
		return err2
	}
	return resourceBaiduCloudCCEv2InstanceGroupReplicaRead(d, meta)
}

func resourceBaiduCloudCCEv2InstanceGroupReplicaDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
