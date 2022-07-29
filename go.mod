module github.com/terraform-providers/terraform-provider-baiducloud

replace github.com/baidubce/bce-sdk-go => /home/wangzhihu.linux/work/bce-sdk-go/

require (
	github.com/baidubce/bce-sdk-go v0.9.128
	github.com/hashicorp/terraform-plugin-sdk v1.17.2
	github.com/mitchellh/go-homedir v1.1.0
)

go 1.11
