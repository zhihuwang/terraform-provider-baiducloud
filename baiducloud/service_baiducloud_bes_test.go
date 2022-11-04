package baiducloud

import (
	"encoding/json"
	"log"
	"os"
	"testing"

	"github.com/baidubce/bce-sdk-go/services/bes"
	"github.com/terraform-providers/terraform-provider-baiducloud/baiducloud/connectivity"
)

func TestDelBes(t *testing.T) {
	config := getConfig()
	client, err := config.Client()
	log.Printf("err=%v", err)
	raw, err := client.WithBesClient(func(client *bes.Client) (interface{}, error) {
		return client.DeleteCluster(&bes.GetESClusterRequest{
			ClusterId: "xxxxx",
		})
	})
	log.Printf("err=%v", err)
	result := raw.(*bes.DeleteESClusterResponse)
	data, _ := json.MarshalIndent(result, " ", " ")
	log.Printf("create_res=%v", string(data))

}
func TestGetBes(t *testing.T) {
	config := getConfig()
	client, err := config.Client()
	log.Printf("err=%v", err)
	raw, err := client.WithBesClient(func(client *bes.Client) (interface{}, error) {
		return client.GetCluster(&bes.GetESClusterRequest{
			ClusterId: "xxxx",
		})
	})
	log.Printf("err=%v", err)
	result := raw.(*bes.DetailESClusterResponse)
	data, _ := json.MarshalIndent(result, " ", " ")
	log.Printf("create_res=%v", string(data))

}
func getConfig() connectivity.Config {
	return connectivity.Config{
		AccessKey: os.Getenv("BAIDU_ACCESSS_KEY"),
		SecretKey: os.Getenv("BAIDU_SECRET_KEY"),
		Region:    connectivity.DefaultRegion,
	}
}
func TestBes(t *testing.T) {
	config := getConfig()
	client, err := config.Client()
	log.Printf("err=%v", err)
	modules := make([]*bes.ESClusterModule, 0)
	em := &bes.ESClusterModule{
		InstanceNum: 3,
		SlotType:    "bes.g3.c2m8",
		DiskSlotInfo: &bes.ESDiskSlotInfo{
			Size: 20,
			Type: "premium_ssd",
		},
		Type: "es_dedicated_master",
	}
	modules = append(modules, em)
	em1 := &bes.ESClusterModule{
		InstanceNum: 1,
		SlotType:    "bes.g3.c2m8",
		DiskSlotInfo: &bes.ESDiskSlotInfo{
			Size: 60,
			Type: "premium_ssd",
		},
		Type: "es_node",
	}
	modules = append(modules, em1)
	em2 := &bes.ESClusterModule{
		InstanceNum: 1,
		SlotType:    "bes.c3.c1m2",
		Type:        "kibana",
	}
	modules = append(modules, em2)
	args := &bes.ESClusterRequest{
		Name:            "xxxx-log-es",
		SecurityGroupID: "g-xxxxx",
		SubnetUUID:      "sbn-xxxx",
		AvailableZone:   "cn-bj-a",
		VpcID:           "vpc-xxxx",
		IsOldPackage:    false,
		Password:        "xxxxxxxx",
		Version:         "7.4.2",
		Billing: bes.ESBilling{
			PaymentType: "postpay",
		},
		Modules: modules,
	}
	argdata, _ := json.MarshalIndent(*args, " ", " ")
	log.Printf("request=%v", string(argdata))
	raw, err := client.WithBesClient(func(client *bes.Client) (interface{}, error) {
		return client.CreateCluster(args)
	})
	log.Printf("err=%v", err)
	data, _ := json.MarshalIndent(raw, " ", " ")
	log.Printf("create_res=%v", string(data))
}