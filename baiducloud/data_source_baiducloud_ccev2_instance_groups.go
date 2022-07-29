/*
Use this data source to list instancegroups .

Example Usage

```hcl
data "baiducloud_ccev2_instance_groups" "default" {
  cluster_id = baiducloud_ccev2_cluster.default_custom.id
  page_no = 0
  page_size = 0
}
```
*/
package baiducloud

import (
	"errors"
	"log"

	ccev2 "github.com/baidubce/bce-sdk-go/services/cce/v2"
	"github.com/baidubce/bce-sdk-go/services/cce/v2/types"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/terraform-providers/terraform-provider-baiducloud/baiducloud/connectivity"
)

func dataSourceBaiduCloudCCEv2ClusterInstanceGroups() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceBaiduCloudCCEv2InstanceGroupsRead,
		Schema: map[string]*schema.Schema{
			//Query Params
			"cluster_id": {
				Type:        schema.TypeString,
				Description: "CCEv2 Cluster ID",
				Required:    true,
				ForceNew:    true,
			},
			"page_no": {
				Type:        schema.TypeInt,
				Description: "Page number of query result",
				Optional:    true,
				ForceNew:    true,
				Default:     0,
			},
			"page_size": {
				Type:        schema.TypeInt,
				Description: "The size of every page",
				Optional:    true,
				ForceNew:    true,
				Default:     0,
			},
			"left_nodes": {
				Type:        schema.TypeInt,
				Description: "The left nodes the repica group",
				Optional:    true,
				ForceNew:    true,
				Default:     0,
			},
			//Query Result
			"total_count": {
				Type:        schema.TypeInt,
				Description: "The total count of the result",
				Computed:    true,
			},
			"instance_group_list": {
				Type:        schema.TypeList,
				Description: "The search result",
				Computed:    true,
				Elem:        resourceCCEv2InstanceGroup(),
			},
		},
	}
}

func dataSourceBaiduCloudCCEv2InstanceGroupsRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*connectivity.BaiduClient)
	args := &ccev2.ListInstanceGroupsArgs{}
	listOpts := &ccev2.InstanceGroupListOption{}
	if value, ok := d.GetOk("cluster_id"); ok && value.(string) != "" {
		args.ClusterID = value.(string)
	} else {
		err := errors.New("get cluster_id fail or cluster_id empty")
		log.Printf("Build ListInstanceByInstanceGroupIDArgs Error:" + err.Error())
		return WrapError(err)
	}
	if value, ok := d.GetOk("page_size"); ok {
		listOpts.PageSize = value.(int)
	}
	left_nodes := 0
	if value, ok := d.GetOk("left_nodes"); ok {
		left_nodes = value.(int)
	}
	if value, ok := d.GetOk("page_no"); ok {
		listOpts.PageNo = value.(int)
	}
	args.ListOption = listOpts

	action := "Get CCEv2 InstanceGroups by Cluster ID:" + args.ClusterID
	raw, err := client.WithCCEv2Client(func(client *ccev2.Client) (i interface{}, e error) {
		return client.ListInstanceGroups(args)
	})
	if err != nil {
		log.Printf("List InstanceGroup Instances Error:" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_instance_groups", action, BCESDKGoERROR)
	}
	addDebug(action, raw)

	response := raw.(*ccev2.ListInstanceGroupResponse)
	if response.Page.List == nil {
		err := errors.New("instance list is nil")
		log.Printf("List InstanceGroup Instances Error:" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_instance_groups", action, BCESDKGoERROR)
	}

	groups, err := convertInstanceGroupFromJsonToMap(response.Page.List, types.ClusterRoleNode, left_nodes)
	if err != nil {
		log.Printf("Get Instance Group Fail" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_instance_groups", action, BCESDKGoERROR)
	}

	err = d.Set("instance_group_list", groups)
	if err != nil {
		log.Printf("Set 'instance_list' to State Error:" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_instance_groups", action, BCESDKGoERROR)
	}

	err = d.Set("total_count", response.Page.TotalCount)
	if err != nil {
		log.Printf("Set 'total_count' to State Error:" + err.Error())
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_instance_groups", action, BCESDKGoERROR)
	}

	d.SetId(resource.UniqueId())

	return nil
}
