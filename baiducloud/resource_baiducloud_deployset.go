/*
Provide a resource to create an deployset.

Example Usage

```hcl
resource "baiducloud_deployset" "default" {
  name              = "test"
  desc = "test deploy"
  strategy = "HOST_HA"
}
```

Import

deployset can be imported, e.g.

```hcl
$ terraform import baiducloud_deployset.default deployset
```
*/
package baiducloud

import (
	"time"

	"github.com/baidubce/bce-sdk-go/bce"
	"github.com/baidubce/bce-sdk-go/services/bcc/api"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"

	"github.com/terraform-providers/terraform-provider-baiducloud/baiducloud/connectivity"
)

func resourceBaiduCloudDeployset() *schema.Resource {
	return &schema.Resource{
		Create: resourceBaiduCloudDeploysetCreate,
		Read:   resourceBaiduCloudDeploysetRead,
		Update: resourceBaiduCloudDeploysetUpdate,
		Delete: resourceBaiduCloudDeploysetDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Description:  "Deployset name",
				Computed:     true,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(1, 65),
			},
			"desc": {
				Type:        schema.TypeString,
				Description: "Deployset desc",
				Optional:    true,
				Computed:    true,
			},
			"strategy": {
				Type:        schema.TypeString,
				Description: "Deployset strategy: HOST_HA RACK_HA TOR_HA",
				Required:    true,
			},
			"concurrency": {
				Type:        schema.TypeInt,
				Description: "concurrency",
				Computed:    true,
			},
			"instance_count": {
				Type:        schema.TypeInt,
				Description: "Instance Count",
				Computed:    true,
			},
			"bcc_instance_cnt": {
				Type:        schema.TypeInt,
				Description: "Deployset bcc instance count",
				Computed:    true,
			},
			"bbc_instance_cnt": {
				Type:        schema.TypeInt,
				Description: "bbc instance count",
				Computed:    true,
			},
			"instance_total": {
				Type:        schema.TypeInt,
				Description: "instance total",
				Computed:    true,
			},
		},
	}
}

func resourceBaiduCloudDeploysetCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.BaiduClient)
	bccService := BccService{client}
	createDeploysetArgs := buildBaiduCloudCreateDeploysetArgs(d)

	action := "Create deployset  " + createDeploysetArgs.Name

	err := resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		response, err := bccService.CreateDeployset(createDeploysetArgs)
		if err != nil {
			if IsExceptedErrors(err, []string{bce.EINTERNAL_ERROR}) {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		d.SetId(response.DeploySetId)
		return nil
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_deployset", action, BCESDKGoERROR)
	}

	return resourceBaiduCloudDeploysetRead(d, meta)
}

func resourceBaiduCloudDeploysetRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.BaiduClient)
	bccClient := BccService{client}

	id := d.Id()

	action := "Query deployset " + id
	result, err := bccClient.GetDeployset(id)
	addDebug(action, result)

	if err != nil {
		if NotFoundError(err) {
			d.SetId("")
			return nil
		}
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_deployset", action, BCESDKGoERROR)
	}

	d.Set("name", result.Name)
	d.Set("desc", result.Desc)
	d.Set("strategy", result.Strategy)
	d.Set("concurrency", result.Concurrency)
	d.Set("instance_count", len(result.InstanceList))
	return nil
}

func resourceBaiduCloudDeploysetUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.BaiduClient)
	bccClient := BccService{client}

	id := d.Id()

	if d.HasChange("name") || d.HasChange("desc") {
		args := &api.ModifyDeploySetArgs{
			ClientToken: buildClientToken(),
		}
		if v, ok := d.GetOk("name"); ok {
			args.Name = v.(string)
		}
		if v, ok := d.GetOk("desc"); ok {
			args.Desc = v.(string)
		}

		if err := bccClient.ModifyDeploySet(id, args); err != nil {
			return WrapError(err)
		}
		d.SetPartial("name")
		d.SetPartial("desc")
	}

	d.Partial(false)
	return resourceBaiduCloudDeploysetRead(d, meta)
}

func resourceBaiduCloudDeploysetDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.BaiduClient)
	bccService := BccService{client: client}

	id := d.Id()
	action := "Delete Deployset " + id

	err := resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		_, errGet := bccService.GetDeployset(id)
		if errGet != nil {
			return resource.NonRetryableError(errGet)
		}

		// if len(detail.InstanceList) > 0 {
		// 	return resource.RetryableError(error.O)
		// }

		errDelete := bccService.DeleteDeploySet(id)
		if errDelete != nil {
			if IsExceptedErrors(errDelete, []string{bce.EINTERNAL_ERROR}) {
				return resource.RetryableError(errDelete)
			}
			return resource.NonRetryableError(errDelete)
		}
		return nil
	})

	if err != nil {
		if IsExceptedErrors(err, ObjectNotFound) {
			return nil
		}
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_deployset", action, BCESDKGoERROR)
	}

	return nil
}

func buildBaiduCloudCreateDeploysetArgs(d *schema.ResourceData) *api.CreateDeploySetArgs {
	request := &api.CreateDeploySetArgs{}

	if v, ok := d.GetOk("name"); ok && v.(string) != "" {
		request.Name = v.(string)
	}

	if v, ok := d.GetOk("desc"); ok {
		request.Desc = v.(string)
	}

	if v, ok := d.GetOk("concurrency"); ok {
		con := v.(int)
		if con > 0 {
			request.Concurrency = con
		}
	}
	if v, ok := d.GetOk("strategy"); ok {
		request.Strategy = v.(string)
	}
	request.ClientToken = buildClientToken()
	return request
}
