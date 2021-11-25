module github.com/terraform-providers/terraform-provider-baiducloud

replace github.com/baidubce/bce-sdk-go => /home/ahoo/work/bce-sdk-go/

require (
	github.com/baidubce/bce-sdk-go v0.9.79
	github.com/hashicorp/terraform v0.12.31
	github.com/mitchellh/go-homedir v1.1.0
)

go 1.11
