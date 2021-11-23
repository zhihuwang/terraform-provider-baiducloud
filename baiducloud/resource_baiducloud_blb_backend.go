/*
Provide a resource to create an blb.

Example Usage

```hcl
resource "baiducloud_blb_backend" "default" {
  blb_id        = "testLoadBalance"

  backend {
    instance_id = "i-12444"
    weight = 50
  }
}
```

Import

blb can be imported, e.g.

```hcl
$ terraform import baiducloud_blb_backend.default id
```
*/
package baiducloud

import (
	"time"

	"github.com/baidubce/bce-sdk-go/bce"
	"github.com/baidubce/bce-sdk-go/services/blb"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"

	"github.com/terraform-providers/terraform-provider-baiducloud/baiducloud/connectivity"
)

func resourceBaiduCloudBLBBackend() *schema.Resource {
	return &schema.Resource{
		Create: resourceBaiduCloudBLBBackendCreate,
		Read:   resourceBaiduCloudBLBBackendRead,
		Update: resourceBaiduCloudBLBBackendUpdate,
		Delete: resourceBaiduCloudBLBBackendDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"blb_id": {
				Type:         schema.TypeString,
				Description:  "LoadBalance instance's id",
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(1, 65),
			},
			"server": {
				Type:        schema.TypeList,
				Description: "backend servers of the blb.",
				MinItems:    1,
				Required:    true,
				MaxItems:    10,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"instance_id": {
							Type:        schema.TypeString,
							Description: "The instance_id of bcc server",
							Required:    true,
						},
						"weight": {
							Type:         schema.TypeInt,
							Description:  "the weight of the bcc server",
							Required:     true,
							ValidateFunc: validation.IntBetween(0, 100),
						},
					},
				},
			},
			"status": {
				Type:        schema.TypeList,
				Description: "all backend servers for blb",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"instance_id": {
							Type:        schema.TypeString,
							Description: "The instance_id of bbc.",
							Optional:    true,
						},
						"weight": {
							Type:        schema.TypeInt,
							Description: "the weight of the server",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func resourceBaiduCloudBLBBackendCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.BaiduClient)
	blbService := BLBService{client}
	blb_id := d.Get("blb_id").(string)
	createArgs := buildBaiduCloudCreateBLBBackendArgs(d)
	action := "add blb backends for " + blb_id

	err := resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		raw, err := client.WithBLBClient(func(blbClient *blb.Client) (i interface{}, e error) {
			return nil, blbClient.AddBackendServers(blb_id, createArgs)
		})
		if err != nil {
			if IsExceptedErrors(err, []string{bce.EINTERNAL_ERROR}) {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}

		addDebug(action, raw)
		d.SetId(blb_id)
		return nil
	})

	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_blb_backend", action, BCESDKGoERROR)
	}

	stateConf := buildStateConf(
		APPBLBProcessingStatus,
		APPBLBAvailableStatus,
		d.Timeout(schema.TimeoutCreate),
		blbService.BLBStateRefreshFunc(d.Id(), APPBLBFailedStatus))
	if _, err := stateConf.WaitForState(); err != nil {
		return WrapError(err)
	}

	return resourceBaiduCloudBLBRead(d, meta)
}
func resourceBaiduCloudBLBBackendRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.BaiduClient)

	blbId := d.Id()
	action := "Query blb " + blbId

	raw, err := client.WithBLBClient(func(blbclient *blb.Client) (i interface{}, e error) {
		return blbclient.DescribeBackendServers(blbId, &blb.DescribeBackendServersArgs{
			MaxKeys: 1000,
		})
	})
	if err != nil {
		if NotFoundError(err) {
			d.SetId("")
			return nil
		}
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_blb_backend", action, BCESDKGoERROR)
	}
	result := raw.(*blb.DescribeBackendServersResult)
	backendList := make([]map[string]interface{}, 0)
	for _, instance := range result.BackendServerList {
		backendList = append(backendList, map[string]interface{}{
			"instance_id": instance.InstanceId,
			"weight":      instance.Weight,
		})
	}
	d.Set("status", backendList)
	if d.Get("server") != nil && len(d.Get("server").([]interface{})) == 0 && len(backendList) > 0 {
		d.Set("server", backendList)
	}

	return nil
}

func resourceBaiduCloudBLBBackendUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.BaiduClient)
	blbService := BLBService{client}

	blbId := d.Id()
	action := "Update blb backend server for " + blbId

	update := false

	if d.HasChange("server") {
		update = true
	}

	stateConf := buildStateConf(
		APPBLBProcessingStatus,
		APPBLBAvailableStatus,
		d.Timeout(schema.TimeoutUpdate),
		blbService.BLBStateRefreshFunc(d.Id(), APPBLBFailedStatus))

	if update {
		d.Partial(true)
		_, err := client.WithBLBClient(func(client *blb.Client) (i interface{}, e error) {
			return nil, client.UpdateBackendServers(blbId, buildBaiduCloudUpdateBLBBackendArgs(d))
		})

		if err != nil {
			if NotFoundError(err) {
				d.SetId("")
				return nil
			}
			return WrapErrorf(err, DefaultErrorMsg, "baiducloud_blb_backend", action, BCESDKGoERROR)
		}

		if _, err := stateConf.WaitForState(); err != nil {
			return WrapError(err)
		}

		d.SetPartial("server")
	}

	d.Partial(false)
	return resourceBaiduCloudBLBRead(d, meta)
}
func resourceBaiduCloudBLBBackendDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.BaiduClient)

	blbId := d.Id()
	action := "Delete blb backend servers for " + blbId

	err := resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		_, err := client.WithBLBClient(func(client *blb.Client) (i interface{}, e error) {
			return blbId, client.RemoveBackendServers(blbId, buildBaiduCloudDeleteBLBBackendArgs(d))
		})
		addDebug(action, blbId)

		if err != nil {
			if IsExceptedErrors(err, []string{bce.EINTERNAL_ERROR}) {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}

		return nil
	})

	if err != nil {
		if IsExceptedErrors(err, ObjectNotFound) {
			return nil
		}
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_blb_backend", action, BCESDKGoERROR)
	}

	return nil
}

func buildBaiduCloudCreateBLBBackendArgs(d *schema.ResourceData) *blb.AddBackendServersArgs {
	requestBody := &blb.AddBackendServersArgs{
		ClientToken: buildClientToken(),
	}
	requestBody.BackendServerList = *buildBackendServers(d)
	return requestBody
}

func buildBaiduCloudUpdateBLBBackendArgs(d *schema.ResourceData) *blb.UpdateBackendServersArgs {
	requestBody := &blb.UpdateBackendServersArgs{
		ClientToken: buildClientToken(),
	}
	requestBody.BackendServerList = *buildBackendServers(d)
	return requestBody
}

func buildBaiduCloudDeleteBLBBackendArgs(d *schema.ResourceData) *blb.RemoveBackendServersArgs {
	requestBody := &blb.RemoveBackendServersArgs{
		ClientToken: buildClientToken(),
	}
	list := buildBackendServers(d)
	ids := make([]string, len(*list))
	for idx, item := range *list {
		ids[idx] = item.InstanceId
	}
	requestBody.BackendServerList = ids
	return requestBody
}
func buildBackendServers(d *schema.ResourceData) *[]blb.BackendServerModel {
	backends := d.Get("server").([]interface{})
	serverList := make([]blb.BackendServerModel, len(backends))
	for index, bk := range backends {
		bkMap := bk.(map[string]interface{})
		if bkMap["instance_id"] != nil {
			serverList[index] = blb.BackendServerModel{
				InstanceId: bkMap["instance_id"].(string),
				Weight:     bkMap["weight"].(int),
			}
		}
	}
	return &serverList
}
