package baiducloud

import (
	"github.com/baidubce/bce-sdk-go/services/bes"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/terraform-providers/terraform-provider-baiducloud/baiducloud/connectivity"
)

type BesService struct {
	client *connectivity.BaiduClient
}

func (s *BesService) ClusterStateRefreshes(clusterId string, failState []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		action := "Query BES Cluster " + clusterId
		raw, err := s.client.WithBesClient(func(besClient *bes.Client) (i interface{}, e error) {
			return besClient.GetCluster(&bes.GetESClusterRequest{
				ClusterId: clusterId,
			})
		})
		addDebug(action, raw)
		if err != nil {
			if NotFoundError(err) {
				return 0, string("Deleted"), nil
			}
			return nil, "", WrapErrorf(err, DefaultErrorMsg, "baiducloud_es_cluster", action, BCESDKGoERROR)
		}
		if raw == nil {
			return nil, "", nil
		}

		result := raw.(*bes.DetailESClusterResponse)
		for _, statue := range failState {
			if result.Result.ActualStatus == statue {
				return result, string(result.Result.ActualStatus), WrapError(Error(GetFailTargetStatus, result.Result.ActualStatus))
			}
		}

		addDebug(action, raw)
		return result, string(result.Result.ActualStatus), nil
	}
}

func resourceBESClusterModule() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"instance_num": {
				Type:        schema.TypeInt,
				Description: "instance num",
				Optional:    true,
				Default:     1,
			},
			"slot_type": {
				Type:        schema.TypeString,
				Description: "slot type, eg: bes.g3.c2m8",
				Required:    true,
			},
			"disk_type": {
				Type:        schema.TypeInt,
				Description: "instance num",
				Optional:    true,
				Default:     "ssd",
			},
			"disk_size": {
				Type:        schema.TypeInt,
				Description: "disk size",
				Optional:    true,
				Default:     0,
			},
		},
	}
}

//===================Build系函数用于将.tf参数构建SDK请求参数并调用===================
//.tf是用户传入的配置文件，某些sdk要求的值可能并没有设置
//Tip: Build系函数对于sdk参数中存在但是.tf中没有设置的参数，会自动跳过赋值，即试用默认值

func buildBESCreateClusterRequest(d *schema.ResourceData) (*bes.ESClusterRequest, error) {

	clusterSpec := &bes.ESClusterRequest{}
	if v, ok := d.GetOk("name"); ok && v.(string) != "" {
		clusterSpec.Name = v.(string)
	}
	if v, ok := d.GetOk("available_zone"); ok && v.(string) != "" {
		clusterSpec.AvailableZone = v.(string)
	}
	if v, ok := d.GetOk("payment_type"); ok && v.(string) != "" {
		if v.(string) == "Postpaid" {
			clusterSpec.Billing.PaymentType = "postpay"
		} else {
			clusterSpec.Billing.PaymentType = "prepay"
		}
		if v, ok := d.GetOk("payment_time"); ok && v.(int) > -1 {
			clusterSpec.Billing.Time = v.(int)
		}
	}
	clusterSpec.IsOldPackage = false

	if v, ok := d.GetOk("security_group_id"); ok && v.(string) != "" {
		clusterSpec.SecurityGroupID = v.(string)
	}
	if v, ok := d.GetOk("subnet_id"); ok && v.(string) != "" {
		clusterSpec.SubnetUUID = v.(string)
	}
	if v, ok := d.GetOk("version"); ok && v.(string) != "" {
		clusterSpec.Version = v.(string)
	}

	if v, ok := d.GetOk("modules"); ok && len(v.([]interface{})) > 0 {
		modules := make([]*bes.ESClusterModule, 0)
		for _, m := range v.([]interface{}) {
			moduleMap := m.(map[string]interface{})
			em := &bes.ESClusterModule{}
			if moduleMap["instance_num"] != nil {
				em.InstanceNum = moduleMap["instance_num"].(int)
			}
			if moduleMap["slot_type"] != nil {
				em.SlotType = moduleMap["slot_type"].(string)
			}
			if moduleMap["disk_type"] != nil {
				em.DiskSlotInfo.Type = moduleMap["disk_type"].(string)
			}
			if moduleMap["disk_size"] != nil {
				em.DiskSlotInfo.Size = moduleMap["disk_size"].(int)
			}
			modules = append(modules, em)
		}
		clusterSpec.Modules = modules
	}

	return clusterSpec, nil
}
