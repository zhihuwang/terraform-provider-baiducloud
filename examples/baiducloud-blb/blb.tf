resource "baiducloud_blb" "default" {
  name        = "blb-test"
  vpc_id      = "vpc-p30tghsn8zqm"
  subnet_id   = "sbn-qnjmaq43e46r"

  tags = {
     env = "test"
     used = "tf-test"
     layer = "dmz"
     group = "devops"
     owner = "wangzhihu"
  }
}

resource "baiducloud_blb_listener" "default" {
  blb_id               = baiducloud_blb.default.id
  listener_port        = 8080
  protocol             = "TCP"
  scheduler            = "LeastConnection"
}

resource "baiducloud_blb_backend" "default" {
  blb_id        = baiducloud_blb.default.id
  backend {
    instance_id = baiducloud_instance.id
    weight = 100
  }
}
data "baiducloud_images" "default" {
    name_regex = trimspace("img_base_prod")
}
data "httpclient_request" "spec" {
  url = "http://op-do-cmdb-api.prod-devops.k8s.chj.cloud/op-do-cmdb-api/v1-0/commondata/md_supplier_instance_type?type_id=60ab67ee109635c34847b147&supplier_id=60950c3b00495e0ea01427cb"
  request_headers = {
    Content-Type: "application/json",
  }
  request_body = ""
}
resource "baiducloud_instance" "default"{
    lifecycle {
            # 不允许apply的时候意外删除虚拟机，除非明确的destory命令
        prevent_destroy = true
        # 忽略一些不可能发生的变更，减少意外判定删除重建
        ignore_changes = [
            tags,
            image_id,
            related_release_flag,
            delete_cds_snapshot_flag,
            timeouts,
            cds_auto_renew,
            cds_disks,
            expire_time,
            instance_type,
            instance_spec,
            ]
    }
    image_id = data.baiducloud_images.default.images[0].id
    name = "op-su-test-7-mp0su"
    availability_zone = "cn-su-c"
    cpu_count = jsondecode(data.httpclient_request.spec.response_body).data[0].cpu_count
    
    memory_capacity_in_gb =jsondecode(data.httpclient_request.spec.response_body).data[0].memory
    
    instance_spec = jsondecode(data.httpclient_request.spec.response_body).data[0].code
    
    # 如果没有值，使用默认N5，使用供应商规格创建之后百度云会返回实际的instance_type
    instance_type ="N5"
    security_groups = split(",","g-15de5u7f9vve,g-u5ybbzzvc75y")
    subnet_id = "sbn-qnjmaq43e46r"
    delete_cds_snapshot_flag = true
    related_release_flag  = true
    
    billing {
        payment_timing = "Postpaid"
    }
    
    root_disk_size_in_gb  = 45
    root_disk_storage_type = "cloud_hp1"
  
    cds_disks{
  	    cds_size_in_gb = "50"
  	    storage_type ="hp1"
    }
  
    tags = {
        env = "test"
        used = "tf-test"
        layer = "dmz"
        group = "devops"
        owner = "wangzhihu"
        cmdb_id = "617214701838b29638fd0f06"
    }
}