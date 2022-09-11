package baiducloud

import (
	"encoding/json"
	"log"
	"os"
	"testing"

	"github.com/baidubce/bce-sdk-go/services/bbc"
	"github.com/terraform-providers/terraform-provider-baiducloud/baiducloud/connectivity"
)

func TestFlavorImage(t *testing.T) {
	config := connectivity.Config{
		AccessKey: os.Getenv("BAIDU_ACCESSS_KEY"),
		SecretKey: os.Getenv("BAIDU_SECRET_KEY"),
		Region:    connectivity.DefaultRegion,
	}
	client, _ := config.Client()
	bbcv2Service := BbcService{client}
	listArgs := &bbc.ListImageArgs{}
	images, err := bbcv2Service.ListAllFlavorImages("BBC-G4-02S", listArgs)
	data, _ := json.MarshalIndent(images, " ", " ")
	log.Printf("err=%v", err)
	log.Printf("images=%s", string(data))
}
func TestListImage(t *testing.T) {
	config := connectivity.Config{
		AccessKey: os.Getenv("BAIDU_ACCESSS_KEY"),
		SecretKey: os.Getenv("BAIDU_SECRET_KEY"),
		Region:    connectivity.DefaultRegion,
	}
	client, _ := config.Client()
	bbcv2Service := BbcService{client}
	listArgs := &bbc.ListImageArgs{}
	images, err := bbcv2Service.ListAllBbcImages(listArgs)
	data, _ := json.MarshalIndent(images, " ", " ")
	log.Printf("err=%v", err)
	log.Printf("images=%s", string(data))
}
