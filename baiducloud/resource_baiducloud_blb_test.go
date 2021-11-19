package baiducloud

import (
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/baidubce/bce-sdk-go/services/blb"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/terraform-providers/terraform-provider-baiducloud/baiducloud/connectivity"
)

const (
	testAccBLBResourceType     = "baiducloud_blb"
	testAccBLBResourceName     = testAccBLBResourceType + "." + BaiduCloudTestResourceName
	testAccBLBResourceAttrName = BaiduCloudTestResourceAttrNamePrefix + "BLB"
)

func init() {
	resource.AddTestSweepers(testAccBLBResourceType, &resource.Sweeper{
		Name: testAccBLBResourceType,
		F:    testSweepBLBs,
	})
}

func testSweepBLBs(region string) error {
	rawClient, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("get BaiduCloud client error: %s", err)
	}
	client := rawClient.(*connectivity.BaiduClient)
	blbService := BLBService{client}

	listArgs := &blb.DescribeLoadBalancersArgs{}
	blbList, _, err := blbService.ListAllBLB(listArgs)
	if err != nil {
		return fmt.Errorf("get BLBs error: %s", err)
	}

	for _, blbModel := range blbList {
		name := blbModel.Name
		blbId := blbModel.BlbId
		if !strings.HasPrefix(name, BaiduCloudTestResourceAttrNamePrefix) {
			log.Printf("[INFO] Skipping BLB: %s (%s)", name, blbId)
			continue
		}

		log.Printf("[INFO] Deleting BLB: %s (%s)", name, blbId)
		_, err := client.WithBLBClient(func(client *blb.Client) (i interface{}, e error) {
			return nil, client.DeleteLoadBalancer(blbId)
		})
		if err != nil {
			log.Printf("[ERROR] Failed to delete BLB %s (%s)", name, blbId)
		}
	}

	return nil
}

//lintignore:AT003
func TestAccBaiduCloudBLB(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccBLBDestory,

		Steps: []resource.TestStep{
			{
				Config: testAccBLBConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaiduCloudDataSourceId(testAccBLBResourceName),
					resource.TestCheckResourceAttr(testAccBLBResourceName, "name", testAccBLBResourceAttrName),
					resource.TestCheckResourceAttr(testAccBLBResourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(testAccBLBResourceName, "cidr", "192.168.0.0/24"),
					resource.TestCheckResourceAttr(testAccBLBResourceName, "subnet_cidr", "192.168.0.0/24"),
					resource.TestCheckResourceAttrSet(testAccBLBResourceName, "create_time"),
					resource.TestCheckResourceAttrSet(testAccBLBResourceName, "vpc_id"),
					resource.TestCheckResourceAttrSet(testAccBLBResourceName, "subnet_id"),
					resource.TestCheckResourceAttrSet(testAccBLBResourceName, "vpc_name"),
					resource.TestCheckResourceAttrSet(testAccBLBResourceName, "subnet_name"),
				),
			},
			{
				ResourceName:      testAccBLBResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccBLBConfigUpdate(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaiduCloudDataSourceId(testAccBLBResourceName),
					resource.TestCheckResourceAttr(testAccBLBResourceName, "name", testAccBLBResourceAttrName+"Update"),
					resource.TestCheckResourceAttr(testAccBLBResourceName, "description", "test update"),
					resource.TestCheckResourceAttr(testAccBLBResourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(testAccBLBResourceName, "cidr", "192.168.0.0/24"),
					resource.TestCheckResourceAttr(testAccBLBResourceName, "subnet_cidr", "192.168.0.0/24"),
					resource.TestCheckResourceAttrSet(testAccBLBResourceName, "create_time"),
					resource.TestCheckResourceAttrSet(testAccBLBResourceName, "vpc_id"),
					resource.TestCheckResourceAttrSet(testAccBLBResourceName, "subnet_id"),
					resource.TestCheckResourceAttrSet(testAccBLBResourceName, "vpc_name"),
					resource.TestCheckResourceAttrSet(testAccBLBResourceName, "subnet_name"),
				),
			},
		},
	})
}

func testAccBLBDestory(s *terraform.State) error {
	client := testAccProvider.Meta().(*connectivity.BaiduClient)
	blbService := BLBService{client}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != testAccBLBResourceType {
			continue
		}

		_, _, err := blbService.GetBLBDetail(rs.Primary.ID)
		if err != nil {
			if NotFoundError(err) {
				continue
			}
			return WrapError(err)
		}
		return WrapError(Error("BLB still exist"))
	}

	return nil
}

func testAccBLBConfig() string {
	return fmt.Sprintf(`
data "baiducloud_specs" "default" {}

data "baiducloud_zones" "default" {}

data "baiducloud_images" "default" {
  image_type = "System"
}

resource "baiducloud_instance" "default" {
  name                  = "%s"
  image_id              = data.baiducloud_images.default.images.0.id
  availability_zone     = data.baiducloud_zones.default.zones.0.zone_name
  cpu_count             = data.baiducloud_specs.default.specs.0.cpu_count
  memory_capacity_in_gb = data.baiducloud_specs.default.specs.0.memory_size_in_gb
  billing = {
    payment_timing = "Postpaid"
  }
}

resource "baiducloud_vpc" "default" {
  name        = "%s"
  description = "test"
  cidr        = "192.168.0.0/24"
}

resource "baiducloud_subnet" "default" {
  name        = "%s"
  zone_name   = data.baiducloud_zones.default.zones.0.zone_name
  cidr        = "192.168.0.0/24"
  vpc_id      = baiducloud_vpc.default.id
  description = "test description"
}

resource "%s" "%s" {
  depends_on  = [baiducloud_instance.default]
  name        = "%s"
  description = ""
  vpc_id      = baiducloud_vpc.default.id
  subnet_id   = baiducloud_subnet.default.id

  tags = {
    "testKey" = "testValue"
  }
}
`, BaiduCloudTestResourceAttrNamePrefix+"BCC",
		BaiduCloudTestResourceAttrNamePrefix+"VPC",
		BaiduCloudTestResourceAttrNamePrefix+"Subnet",
		testAccBLBResourceType, BaiduCloudTestResourceName, testAccBLBResourceAttrName)
}

func testAccBLBConfigUpdate() string {
	return fmt.Sprintf(`
data "baiducloud_specs" "default" {}

data "baiducloud_zones" "default" {}

data "baiducloud_images" "default" {
  image_type = "System"
}

resource "baiducloud_instance" "default" {
  name                  = "%s"
  image_id              = data.baiducloud_images.default.images.0.id
  availability_zone     = data.baiducloud_zones.default.zones.0.zone_name
  cpu_count             = data.baiducloud_specs.default.specs.0.cpu_count
  memory_capacity_in_gb = data.baiducloud_specs.default.specs.0.memory_size_in_gb
  billing = {
    payment_timing = "Postpaid"
  }
}

resource "baiducloud_vpc" "default" {
  name        = "%s"
  description = "test"
  cidr        = "192.168.0.0/24"
}

resource "baiducloud_subnet" "default" {
  name        = "%s"
  zone_name   = data.baiducloud_zones.default.zones.0.zone_name
  cidr        = "192.168.0.0/24"
  vpc_id      = baiducloud_vpc.default.id
  description = "test description"
}

resource "baiducloud_eip" "default" {
  name              = "%s"
  bandwidth_in_mbps = 1
  payment_timing    = "Postpaid"
  billing_method    = "ByTraffic"
}

resource "%s" "%s" {
  depends_on  = [baiducloud_instance.default]
  name        = "%s"
  description = "test update"
  vpc_id      = baiducloud_vpc.default.id
  subnet_id   = baiducloud_subnet.default.id

  tags = {
    "testKey" = "testValue"
  }
}

resource "baiducloud_eip_association" "default" {
  eip           = baiducloud_eip.default.id
  instance_type = "BLB"
  instance_id   = %s.%s.id
}
`, BaiduCloudTestResourceAttrNamePrefix+"BCC",
		BaiduCloudTestResourceAttrNamePrefix+"VPC",
		BaiduCloudTestResourceAttrNamePrefix+"Subnet",
		BaiduCloudTestResourceAttrNamePrefix+"EIP",
		testAccBLBResourceType, BaiduCloudTestResourceName, testAccBLBResourceAttrName+"Update",
		testAccBLBResourceType, BaiduCloudTestResourceName)
}
