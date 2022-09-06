package baiducloud

import (
	"log"
	"testing"

	"github.com/terraform-providers/terraform-provider-baiducloud/baiducloud/connectivity"
)

func testV2Group(t *testing.T) {
	config := connectivity.Config{
		AccessKey: "",
		SecretKey: "",
		Region:    "bj",
	}
	client, err := config.Client()
	ccev2Service := Ccev2Service{client}
	groups, err := ccev2Service.GetInstanceGroupList("cce-**", 1)
	log.Printf("err=%v", err)
	log.Printf("groups=%v", groups)
}
