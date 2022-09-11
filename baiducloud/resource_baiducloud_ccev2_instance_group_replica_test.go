package baiducloud

import (
	"encoding/json"
	"log"
	"os"
	"testing"

	"github.com/terraform-providers/terraform-provider-baiducloud/baiducloud/connectivity"
)

func testV2Group(t *testing.T) {
	config := connectivity.Config{
		AccessKey: os.Getenv("BAIDU_ACCESSS_KEY"),
		SecretKey: os.Getenv("BAIDU_SECRET_KEY"),
		Region:    connectivity.DefaultRegion,
	}
	client, _ := config.Client()
	ccev2Service := Ccev2Service{client}
	groups, err := ccev2Service.GetInstanceGroupList("cce-**", 1)
	log.Printf("err=%v", err)
	data, _ := json.MarshalIndent(groups, " ", " ")
	log.Printf("groups=%s", string(data))
}
