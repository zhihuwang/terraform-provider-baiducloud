/*
Provide a resource to create an blb Listener.

Example Usage

```hcl
[TCP/UDP] Listener
resource "baiducloud_blb_listener" "default" {
  blb_id               = "lb-0d29a3f6"
  listener_port        = 124
  protocol             = "TCP"
  scheduler            = "LeastConnection"
}

[HTTP] Listener
resource "baiducloud_blb_listener" "default" {
  blb_id        = "lb-0d29a3f6"
  listener_port = 129
  protocol      = "HTTP"
  scheduler     = "RoundRobin"
  keep_session  = true
}

[HTTPS] Listener
resource "baiducloud_blb_listener" "default" {
  blb_id               = "lb-0d29a3f6"
  listener_port        = 130
  protocol             = "HTTPS"
  scheduler            = "LeastConnection"
  keep_session         = true
  cert_ids             = ["cert-xvysj80uif1y"]
  encryption_protocols = ["sslv3", "tlsv10", "tlsv11"]
  encryption_type      = "userDefind"
}

[SSL] Listener
resource "baiducloud_blb_listener" "default" {
  blb_id               = "lb-0d29a3f6"
  listener_port        = 131
  protocol             = "SSL"
  scheduler            = "LeastConnection"
  cert_ids             = ["cert-xvysj80uif1y"]
  encryption_protocols = ["sslv3", "tlsv10", "tlsv11"]
  encryption_type      = "userDefind"
}
```
*/
package baiducloud

import (
	"fmt"
	"strconv"
	"time"

	"github.com/baidubce/bce-sdk-go/bce"
	"github.com/baidubce/bce-sdk-go/services/blb"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"

	"github.com/terraform-providers/terraform-provider-baiducloud/baiducloud/connectivity"
)

func resourceBaiduCloudBLBListener() *schema.Resource {
	return &schema.Resource{
		Create: resourceBaiduCloudBLBListenerCreate,
		Read:   resourceBaiduCloudBLBListenerRead,
		Update: resourceBaiduCloudBLBListenerUpdate,
		Delete: resourceBaiduCloudBLBListenerDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"blb_id": {
				Type:        schema.TypeString,
				Description: "ID of the Application LoadBalance instance",
				Required:    true,
				ForceNew:    true,
			},
			"listener_port": {
				Type:         schema.TypeInt,
				Description:  "Listening port, range from 1-65535",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validatePort(),
			},
			"backend_port": {
				Type:         schema.TypeInt,
				Description:  "backend port, range from 1-65535",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validatePort(),
			},
			"protocol": {
				Type:         schema.TypeString,
				Description:  "Listening protocol, support TCP/UDP/HTTP/HTTPS/SSL",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{TCP, UDP, HTTP, HTTPS, SSL}, false),
			},
			"scheduler": {
				Type:         schema.TypeString,
				Description:  "Load balancing algorithm, support RoundRobin/LeastConnection/Hash, if protocol is HTTP/HTTPS, only support RoundRobin/LeastConnection",
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"RoundRobin", "LeastConnection", "Hash"}, false),
			},
			"tcp_session_timeout": {
				Type:         schema.TypeInt,
				Description:  "TCP Listener connection session timeout time(second), default 900, support 10-4000",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IntBetween(10, 4000),
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if v, ok := d.Get("protocol").(string); ok {
						return v != TCP
					}
					return true
				},
			},
			"health_check_timeout_in_second": {
				Type:        schema.TypeInt,
				Description: "health_check_timeout_in_second",
				Optional:    true,
			},
			"health_check_interval": {
				Type:        schema.TypeInt,
				Description: "health_check_interval",
				Optional:    true,
			},
			"unhealthy_threshold": {
				Type:        schema.TypeInt,
				Description: "unhealthy_threshold",
				Optional:    true,
			},
			"healthy_threshold": {
				Type:        schema.TypeInt,
				Description: "healthy_threshold",
				Optional:    true,
			},
			// udp only
			"health_check_string": {
				Type:        schema.TypeString,
				Description: "health_check_string",
				Optional:    true,
			},
			// http or https
			"health_check_port": {
				Type:        schema.TypeInt,
				Description: "health_check_port",
				Optional:    true,
			},
			// http or https
			"health_check_uri": {
				Type:        schema.TypeInt,
				Description: "health_check_port",
				Optional:    true,
			},
			// http or https
			"health_check_normal_status": {
				Type:        schema.TypeInt,
				Description: "health_check_port",
				Optional:    true,
			},
			// http & https
			"keep_session": {
				Type:             schema.TypeBool,
				Description:      "KeepSession or not",
				Optional:         true,
				Computed:         true,
				DiffSuppressFunc: appBlbProtocolTCPUDPSSLSuppressFunc,
			},
			//http & https
			"keep_session_type": {
				Type:             schema.TypeString,
				Description:      "KeepSessionType option, support insert/rewrite, default insert",
				Optional:         true,
				Computed:         true,
				ValidateFunc:     validation.StringInSlice([]string{"insert", "rewrite"}, false),
				DiffSuppressFunc: appBlbProtocolTCPUDPSSLSuppressFunc,
			},
			// http & https
			"keep_session_timeout": {
				Type:             schema.TypeInt,
				Description:      "KeepSession Cookie timeout time(second), support in [1, 15552000], default 3600s",
				Optional:         true,
				Computed:         true,
				ValidateFunc:     validation.IntBetween(1, 15552000),
				DiffSuppressFunc: appBlbProtocolTCPUDPSSLSuppressFunc,
			},
			// http & https
			"keep_session_cookie_name": {
				Type:        schema.TypeString,
				Description: "CookieName which need to covered, useful when keep_session_type is rewrite",
				Optional:    true,
				Computed:    true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					protocolCheck := appBlbProtocolTCPUDPSSLSuppressFunc(k, old, new, d)
					if protocolCheck {
						return true
					}

					if v, ok := d.GetOk("keep_session"); !ok || !(v.(bool)) {
						return true
					}

					if v, ok := d.GetOk("keep_session_type"); ok {
						return v.(string) != "rewrite"
					}

					return true
				},
			},
			// http & https
			"x_forwarded_for": {
				Type:             schema.TypeBool,
				Description:      "Listener xForwardedFor, determine get client real ip or not, default false",
				Optional:         true,
				Computed:         true,
				DiffSuppressFunc: appBlbProtocolTCPUDPSSLSuppressFunc,
			},
			// http & https
			"server_timeout": {
				Type:             schema.TypeInt,
				Description:      "Backend server maximum timeout time, only support in [1, 3600] second, default 30s",
				Optional:         true,
				Computed:         true,
				DiffSuppressFunc: appBlbProtocolTCPUDPSSLSuppressFunc,
				ValidateFunc:     validation.IntBetween(1, 3600),
			},
			// http
			"redirect_port": {
				Type:        schema.TypeInt,
				Description: "Redirect HTTP request to HTTPS Listener, HTTPS Listener port set by this parameter",
				Optional:    true,
				Computed:    true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return d.Get("protocol").(string) != HTTP
				},
			},
			// https && ssl
			"cert_ids": {
				Type:        schema.TypeSet,
				Description: "Listener bind certifications",
				Optional:    true,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				DiffSuppressFunc: appBlbProtocolTCPUDPHTTPSuppressFunc,
			},
			// https && ssl
			"encryption_type": {
				Type:             schema.TypeString,
				Description:      "Listener encryption option, support [compatibleIE, incompatibleIE, userDefind]",
				Optional:         true,
				Computed:         true,
				ValidateFunc:     validation.StringInSlice([]string{"compatibleIE", "incompatibleIE", "userDefind"}, false),
				DiffSuppressFunc: appBlbProtocolTCPUDPHTTPSuppressFunc,
			},
			// https && ssl
			"encryption_protocols": {
				Type:        schema.TypeSet,
				Description: "Listener encryption protocol, only useful when encryption_type is userDefind, support [sslv3, tlsv10, tlsv11, tlsv12]",
				Optional:    true,
				Computed:    true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{"sslv3", "tlsv10", "tlsv11", "tlsv12"}, false),
				},
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if v, ok := d.GetOk("encryption_type"); ok {
						return v.(string) != "userDefind"
					}

					return true
				},
			},
			// https && ssl
			"dual_auth": {
				Type:             schema.TypeBool,
				Description:      "Listener open dual authorization or not, default false",
				Optional:         true,
				Computed:         false,
				DiffSuppressFunc: appBlbProtocolTCPUDPHTTPSuppressFunc,
			},
			// https && ssl
			"client_cert_ids": {
				Type:        schema.TypeSet,
				Description: "Listener import cert list, only useful when dual_auth is true",
				Optional:    true,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				DiffSuppressFunc: appBlbProtocolTCPUDPHTTPSuppressFunc,
			},
		},
	}
}

func resourceBaiduCloudBLBListenerCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.BaiduClient)

	blbId := d.Get("blb_id").(string)
	protocol := d.Get("protocol").(string)
	listenerPort := d.Get("listener_port").(int)
	action := fmt.Sprintf("Create blb %s Listener [%s:%d]", blbId, protocol, listenerPort)

	listenerArgs, err := buildBaiduCloudCreateblbListenerArgs(d, meta)
	if err != nil {
		return WrapError(err)
	}

	err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		raw, err := client.WithBLBClient(func(client *blb.Client) (i interface{}, e error) {
			switch protocol {
			case TCP:
				return blbId, client.CreateTCPListener(blbId, listenerArgs.(*blb.CreateTCPListenerArgs))
			case UDP:
				return blbId, client.CreateUDPListener(blbId, listenerArgs.(*blb.CreateUDPListenerArgs))
			case HTTP:
				return blbId, client.CreateHTTPListener(blbId, listenerArgs.(*blb.CreateHTTPListenerArgs))
			case HTTPS:
				return blbId, client.CreateHTTPSListener(blbId, listenerArgs.(*blb.CreateHTTPSListenerArgs))
			case SSL:
				return blbId, client.CreateSSLListener(blbId, listenerArgs.(*blb.CreateSSLListenerArgs))
			default:
				// never run here
				return blbId, fmt.Errorf("unsupport protocol")
			}
		})

		if err != nil {
			if IsExceptedErrors(err, []string{bce.EINTERNAL_ERROR}) {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}

		addDebug(action, raw)
		d.SetId(strconv.Itoa(listenerPort))

		return nil
	})

	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_blb_listener", action, BCESDKGoERROR)
	}

	return resourceBaiduCloudBLBListenerRead(d, meta)
}

func resourceBaiduCloudBLBListenerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.BaiduClient)
	blbService := BLBService{client}

	blbId := d.Get("blb_id").(string)
	protocol := d.Get("protocol").(string)
	listenerPort := d.Get("listener_port").(int)
	action := fmt.Sprintf("Query blb %s Listener [%s:%d]", blbId, protocol, listenerPort)

	raw, err := blbService.DescribeListener(blbId, protocol, listenerPort)
	if err != nil {
		d.SetId("")
		return WrapError(err)
	}
	addDebug(action, raw)

	switch protocol {
	case HTTP:
		listenerMeta := raw.(*blb.HTTPListenerModel)
		d.Set("scheduler", listenerMeta.Scheduler)
		d.Set("keep_session", listenerMeta.KeepSession)
		d.Set("keep_session_type", listenerMeta.KeepSessionType)
		d.Set("keep_session_timeout", listenerMeta.KeepSessionDuration)
		d.Set("keep_session_cookie_name", listenerMeta.KeepSessionCookieName)
		d.Set("x_forwarded_for", listenerMeta.XForwardedFor)
		d.Set("server_timeout", listenerMeta.ServerTimeout)
		d.Set("redirect_port", listenerMeta.RedirectPort)
		d.Set("listener_port", listenerMeta.ListenerPort)
		d.Set("backend_port", listenerMeta.BackendPort)
	case HTTPS:
		listenerMeta := raw.(*blb.HTTPSListenerModel)
		d.Set("scheduler", listenerMeta.Scheduler)
		d.Set("keep_session", listenerMeta.KeepSession)
		d.Set("keep_session_type", listenerMeta.KeepSessionType)
		d.Set("keep_session_timeout", listenerMeta.KeepSessionDuration)
		d.Set("keep_session_cookie_name", listenerMeta.KeepSessionCookieName)
		d.Set("x_forwarded_for", listenerMeta.XForwardedFor)
		d.Set("server_timeout", listenerMeta.ServerTimeout)
		d.Set("cert_ids", listenerMeta.CertIds)
		d.Set("dual_auth", listenerMeta.DualAuth)
		d.Set("client_cert_ids", listenerMeta.ClientCertIds)
		d.Set("listener_port", listenerMeta.ListenerPort)
		d.Set("backend_port", listenerMeta.BackendPort)
	case SSL:
		listenerMeta := raw.(*blb.SSLListenerModel)
		d.Set("scheduler", listenerMeta.Scheduler)
		d.Set("cert_ids", listenerMeta.CertIds)
		d.Set("encryption_type", listenerMeta.EncryptionType)
		d.Set("encryption_protocols", listenerMeta.EncryptionProtocols)
		d.Set("dual_auth", listenerMeta.DualAuth)
		d.Set("client_cert_ids", listenerMeta.ClientCertIds)
		d.Set("listener_port", listenerMeta.ListenerPort)
		d.Set("backend_port", listenerMeta.BackendPort)
	case TCP:
		listenerMeta := raw.(*blb.TCPListenerModel)
		d.Set("scheduler", listenerMeta.Scheduler)
		d.Set("tcp_session_timeout", listenerMeta.TcpSessionTimeout)
		d.Set("backend_port", listenerMeta.BackendPort)
		d.Set("listener_port", listenerMeta.ListenerPort)
	case UDP:
		listenerMeta := raw.(*blb.UDPListenerModel)
		d.Set("scheduler", listenerMeta.Scheduler)
		d.Set("backend_port", listenerMeta.BackendPort)
		d.Set("listener_port", listenerMeta.ListenerPort)
	default:
		return WrapError(fmt.Errorf("unsupport listener type"))
	}
	d.SetId(strconv.Itoa(listenerPort))

	return nil
}

func resourceBaiduCloudBLBListenerUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.BaiduClient)

	blbId := d.Get("blb_id").(string)
	protocol := d.Get("protocol").(string)
	listenerPort := d.Get("listener_port").(int)
	action := fmt.Sprintf("Update blb %s Listener [%s:%d]", blbId, protocol, listenerPort)

	update, args, err := buildBaiduCloudUpdateBLBListenerArgs(d, meta)
	if err != nil {
		return WrapError(err)
	}
	if update {
		_, err := client.WithBLBClient(func(client *blb.Client) (i interface{}, e error) {
			switch protocol {
			case TCP:
				return nil, client.UpdateTCPListener(blbId, args.(*blb.UpdateTCPListenerArgs))
			case UDP:
				return nil, client.UpdateUDPListener(blbId, args.(*blb.UpdateUDPListenerArgs))
			case HTTP:
				return nil, client.UpdateHTTPListener(blbId, args.(*blb.UpdateHTTPListenerArgs))
			case HTTPS:
				return nil, client.UpdateHTTPSListener(blbId, args.(*blb.UpdateHTTPSListenerArgs))
			case SSL:
				return nil, client.UpdateSSLListener(blbId, args.(*blb.UpdateSSLListenerArgs))
			default:
				return nil, fmt.Errorf("unsupport listener type")
			}
		})
		addDebug(action, nil)

		if err != nil {
			if NotFoundError(err) {
				d.SetId("")
				return nil
			}
			return WrapErrorf(err, DefaultErrorMsg, "baiducloud_blb_listener", action, BCESDKGoERROR)
		}
	}

	return resourceBaiduCloudBLBListenerRead(d, meta)
}

func resourceBaiduCloudBLBListenerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.BaiduClient)

	blbId := d.Get("blb_id").(string)
	protocol := d.Get("protocol").(string)
	listenerPort := d.Get("listener_port").(int)
	action := fmt.Sprintf("Delete blb %s Listener [%s:%d]", blbId, protocol, listenerPort)

	err := resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		_, err := client.WithBLBClient(func(client *blb.Client) (i interface{}, e error) {
			return blbId, client.DeleteListeners(blbId, &blb.DeleteListenersArgs{
				PortList:    []uint16{uint16(listenerPort)},
				ClientToken: buildClientToken(),
			})
		})
		addDebug(action, blbId)

		if err != nil {
			if IsExceptedErrors(err, []string{bce.EINTERNAL_ERROR}) {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}

		return nil
	})

	if err != nil {
		if IsExceptedErrors(err, ObjectNotFound) {
			return nil
		}
		return WrapErrorf(err, DefaultErrorMsg, "baiducloud_blb_listener", action, BCESDKGoERROR)
	}

	return nil
}

func buildBaiduCloudCreateblbListenerArgs(d *schema.ResourceData, meta interface{}) (interface{}, error) {
	protocol := d.Get("protocol").(string)

	switch protocol {
	case TCP:
		args := &blb.CreateTCPListenerArgs{
			ListenerPort: uint16(d.Get("listener_port").(int)),
			BackendPort:  uint16(d.Get("backend_port").(int)),
			Scheduler:    d.Get("scheduler").(string),
			ClientToken:  buildClientToken(),
		}
		if v, ok := d.GetOk("health_check_timeout_in_second"); ok {
			args.HealthCheckTimeoutInSecond = v.(int)
		}
		if v, ok := d.GetOk("health_check_interval"); ok {
			args.HealthCheckInterval = v.(int)
		}
		if v, ok := d.GetOk("unhealthy_threshold"); ok {
			args.UnhealthyThreshold = v.(int)
		}
		if v, ok := d.GetOk("healthy_threshold"); ok {
			args.HealthyThreshold = v.(int)
		}
		if v, ok := d.GetOk("tcp_session_timeout"); ok {
			args.TcpSessionTimeout = v.(int)
		}

		return args, nil
	case UDP:
		args := &blb.CreateUDPListenerArgs{
			ListenerPort: uint16(d.Get("listener_port").(int)),
			Scheduler:    d.Get("scheduler").(string),
			ClientToken:  buildClientToken(),
			BackendPort:  uint16(d.Get("backend_port").(int)),
		}
		if v, ok := d.GetOk("health_check_timeout_in_second"); ok {
			args.HealthCheckTimeoutInSecond = v.(int)
		}
		if v, ok := d.GetOk("health_check_interval"); ok {
			args.HealthCheckInterval = v.(int)
		}
		if v, ok := d.GetOk("unhealthy_threshold"); ok {
			args.UnhealthyThreshold = v.(int)
		}
		if v, ok := d.GetOk("healthy_threshold"); ok {
			args.HealthyThreshold = v.(int)
		}
		if v, ok := d.GetOk("health_check_string"); ok {
			args.HealthCheckString = v.(string)
		}
		return args, nil
	case HTTP:
		return buildBaiduCloudCreateblbHTTPListenerArgs(d, meta)
	case HTTPS:
		return buildBaiduCloudCreateblbHTTPSListenerArgs(d, meta)
	case SSL:
		return buildBaiduCloudCreateblbSSLListenerArgs(d, meta)
	default:
		// never run here
		return nil, fmt.Errorf("listener only support protocol [TCP, UDP, HTTP, HTTPS, SSL], but now set: %s", protocol)
	}
}

func buildBaiduCloudCreateblbHTTPListenerArgs(d *schema.ResourceData, meta interface{}) (*blb.CreateHTTPListenerArgs, error) {
	result := &blb.CreateHTTPListenerArgs{
		ClientToken:  buildClientToken(),
		ListenerPort: uint16(d.Get("listener_port").(int)),
		BackendPort:  uint16(d.Get("backend_port").(int)),
		Scheduler:    d.Get("scheduler").(string),
	}
	if result.Scheduler != "RoundRobin" && result.Scheduler != "LeastConnection" {
		return nil, fmt.Errorf("HTTP Listener scheduler only support [RoundRobin, LeastConnection], but you set: %s", result.Scheduler)
	}
	if v, ok := d.GetOk("health_check_timeout_in_second"); ok {
		result.HealthCheckTimeoutInSecond = v.(int)
	}
	if v, ok := d.GetOk("health_check_interval"); ok {
		result.HealthCheckInterval = v.(int)
	}
	if v, ok := d.GetOk("unhealthy_threshold"); ok {
		result.UnhealthyThreshold = v.(int)
	}
	if v, ok := d.GetOk("healthy_threshold"); ok {
		result.HealthyThreshold = v.(int)
	}
	if v, ok := d.GetOk("health_check_uri"); ok {
		result.HealthCheckURI = v.(string)
	}
	if v, ok := d.GetOk("health_check_port"); ok {
		result.HealthCheckPort = uint16(v.(int))
	}
	if v, ok := d.GetOk("health_check_normal_status"); ok {
		result.HealthCheckNormalStatus = v.(string)
	}

	if v, ok := d.GetOk("keep_session"); ok {
		result.KeepSession = v.(bool)
	}

	if v, ok := d.GetOk("keep_session_type"); ok {
		result.KeepSessionType = v.(string)
	}

	if v, ok := d.GetOk("keep_session_timeout"); ok {
		result.KeepSessionDuration = v.(int)
	}

	if v, ok := d.GetOk("keep_session_cookie_name"); ok {
		result.KeepSessionCookieName = v.(string)
	}

	if v, ok := d.GetOk("x_forwarded_for"); ok {
		result.XForwardedFor = v.(bool)
	}

	if v, ok := d.GetOk("server_timeout"); ok {
		result.ServerTimeout = v.(int)
	}

	if v, ok := d.GetOk("redirect_port"); ok {
		result.RedirectPort = uint16(v.(int))
	}

	return result, nil
}

func buildBaiduCloudCreateblbHTTPSListenerArgs(d *schema.ResourceData, meta interface{}) (*blb.CreateHTTPSListenerArgs, error) {
	result := &blb.CreateHTTPSListenerArgs{
		ClientToken:  buildClientToken(),
		ListenerPort: uint16(d.Get("listener_port").(int)),
		Scheduler:    d.Get("scheduler").(string),
	}
	if result.Scheduler != "RoundRobin" && result.Scheduler != "LeastConnection" {
		return nil, fmt.Errorf("HTTPS Listener scheduler only support [RoundRobin, LeastConnection], but you set: %s", result.Scheduler)
	}

	if v, ok := d.GetOk("health_check_timeout_in_second"); ok {
		result.HealthCheckTimeoutInSecond = v.(int)
	}
	if v, ok := d.GetOk("health_check_interval"); ok {
		result.HealthCheckInterval = v.(int)
	}
	if v, ok := d.GetOk("unhealthy_threshold"); ok {
		result.UnhealthyThreshold = v.(int)
	}
	if v, ok := d.GetOk("healthy_threshold"); ok {
		result.HealthyThreshold = v.(int)
	}
	if v, ok := d.GetOk("health_check_uri"); ok {
		result.HealthCheckURI = v.(string)
	}
	if v, ok := d.GetOk("health_check_port"); ok {
		result.HealthCheckPort = uint16(v.(int))
	}
	if v, ok := d.GetOk("health_check_normal_status"); ok {
		result.HealthCheckNormalStatus = v.(string)
	}

	if v, ok := d.GetOk("keep_session"); ok {
		result.KeepSession = v.(bool)
	}

	if v, ok := d.GetOk("keep_session_type"); ok {
		result.KeepSessionType = v.(string)
	}

	if v, ok := d.GetOk("keep_session_timeout"); ok {
		result.KeepSessionDuration = v.(int)
	}

	if v, ok := d.GetOk("keep_session_cookie_name"); ok {
		result.KeepSessionCookieName = v.(string)
	}

	if v, ok := d.GetOk("x_forwarded_for"); ok {
		result.XForwardedFor = v.(bool)
	}

	if v, ok := d.GetOk("server_timeout"); ok {
		result.ServerTimeout = v.(int)
	}

	if v, ok := d.GetOk("cert_ids"); ok {
		for _, id := range v.(*schema.Set).List() {
			result.CertIds = append(result.CertIds, id.(string))
		}
	}
	if len(result.CertIds) <= 0 {
		return nil, fmt.Errorf("HTTPS Listener require cert, but not set")
	}

	if v, ok := d.GetOk("encryption_type"); ok {
		result.EncryptionType = v.(string)
	}

	if v, ok := d.GetOk("encryption_protocols"); ok {
		for _, p := range v.(*schema.Set).List() {
			result.EncryptionProtocols = append(result.EncryptionProtocols, p.(string))
		}
	}

	if v, ok := d.GetOk("dual_auth"); ok {
		result.DualAuth = v.(bool)
	}

	if v, ok := d.GetOk("client_cert_ids"); ok {
		for _, id := range v.(*schema.Set).List() {
			result.ClientCertIds = append(result.ClientCertIds, id.(string))
		}
	}

	return result, nil
}

func buildBaiduCloudCreateblbSSLListenerArgs(d *schema.ResourceData, meta interface{}) (*blb.CreateSSLListenerArgs, error) {
	result := &blb.CreateSSLListenerArgs{
		ClientToken:  buildClientToken(),
		ListenerPort: uint16(d.Get("listener_port").(int)),
		BackendPort:  uint16(d.Get("backend_port").(int)),
		Scheduler:    d.Get("scheduler").(string),
	}
	if v, ok := d.GetOk("health_check_timeout_in_second"); ok {
		result.HealthCheckTimeoutInSecond = v.(int)
	}
	if v, ok := d.GetOk("health_check_interval"); ok {
		result.HealthCheckInterval = v.(int)
	}
	if v, ok := d.GetOk("unhealthy_threshold"); ok {
		result.UnhealthyThreshold = v.(int)
	}
	if v, ok := d.GetOk("healthy_threshold"); ok {
		result.HealthyThreshold = v.(int)
	}
	if v, ok := d.GetOk("cert_ids"); ok {
		for _, id := range v.(*schema.Set).List() {
			result.CertIds = append(result.CertIds, id.(string))
		}
	}
	if len(result.CertIds) <= 0 {
		return nil, fmt.Errorf("SSL Listener require cert, but not set")
	}

	if v, ok := d.GetOk("encryption_type"); ok {
		result.EncryptionType = v.(string)
	}

	if v, ok := d.GetOk("encryption_protocols"); ok {
		for _, p := range v.(*schema.Set).List() {
			result.EncryptionProtocols = append(result.EncryptionProtocols, p.(string))
		}
	}

	if v, ok := d.GetOk("dual_auth"); ok {
		result.DualAuth = v.(bool)
	}

	if v, ok := d.GetOk("client_cert_ids"); ok {
		for _, id := range v.(*schema.Set).List() {
			result.ClientCertIds = append(result.ClientCertIds, id.(string))
		}
	}

	return result, nil
}

func buildBaiduCloudUpdateBLBListenerArgs(d *schema.ResourceData, meta interface{}) (bool, interface{}, error) {
	protocol := d.Get("protocol").(string)

	switch protocol {
	case TCP:
		if d.HasChange("scheduler") || d.HasChange("tcp_session_timeout") || d.HasChange("health_check_timeout_in_second") || d.HasChange("health_check_interval") || d.HasChange("tcp_session_timeout") || d.HasChange("unhealthy_threshold") || d.HasChange("healthy_threshold") {
			args := &blb.UpdateTCPListenerArgs{
				TcpSessionTimeout: d.Get("tcp_session_timeout").(int),
				ListenerPort:      uint16(d.Get("listener_port").(int)),
				BackendPort:       uint16(d.Get("backend_port").(int)),
				Scheduler:         d.Get("scheduler").(string),
				ClientToken:       buildClientToken(),
			}
			if v, ok := d.GetOk("health_check_timeout_in_second"); ok {
				args.HealthCheckTimeoutInSecond = v.(int)
			}
			if v, ok := d.GetOk("health_check_interval"); ok {
				args.HealthCheckInterval = v.(int)
			}
			if v, ok := d.GetOk("unhealthy_threshold"); ok {
				args.UnhealthyThreshold = v.(int)
			}
			if v, ok := d.GetOk("healthy_threshold"); ok {
				args.HealthyThreshold = v.(int)
			}
			if v, ok := d.GetOk("tcp_session_timeout"); ok {
				args.TcpSessionTimeout = v.(int)
			}
			return true, args, nil
		}
		return false, nil, nil
	case UDP:
		if d.HasChange("scheduler") || d.HasChange("listener_port") || d.HasChange("backend_port") || d.HasChange("health_check_timeout_in_second") || d.HasChange("health_check_interval") || d.HasChange("unhealthy_threshold") || d.HasChange("healthy_threshold") || d.HasChange("health_check_string") {
			args := &blb.UpdateUDPListenerArgs{
				ListenerPort: uint16(d.Get("listener_port").(int)),
				Scheduler:    d.Get("scheduler").(string),
				BackendPort:  uint16(d.Get("backend_port").(int)),
				ClientToken:  buildClientToken(),
			}
			if v, ok := d.GetOk("health_check_timeout_in_second"); ok {
				args.HealthCheckTimeoutInSecond = v.(int)
			}
			if v, ok := d.GetOk("health_check_interval"); ok {
				args.HealthCheckInterval = v.(int)
			}
			if v, ok := d.GetOk("unhealthy_threshold"); ok {
				args.UnhealthyThreshold = v.(int)
			}
			if v, ok := d.GetOk("healthy_threshold"); ok {
				args.HealthyThreshold = v.(int)
			}
			if v, ok := d.GetOk("health_check_string"); ok {
				args.HealthCheckString = v.(string)
			}
			return true, args, nil
		}
		return false, nil, nil
	case HTTP:
		return buildBaiduCloudUpdateBLBHTTPListenerArgs(d, meta)
	case HTTPS:
		return buildBaiduCloudUpdateBLBHTTPSListenerArgs(d, meta)
	case SSL:
		return buildBaiduCloudUpdateBLBSSLListenerArgs(d, meta)
	default:
		// never run here
		return false, nil, fmt.Errorf("listener only support protocol [TCP, UDP, HTTP, HTTPS, SSL], but now set: %s", protocol)
	}
}

func buildBaiduCloudUpdateBLBHTTPListenerArgs(d *schema.ResourceData, meta interface{}) (bool, *blb.UpdateHTTPListenerArgs, error) {
	update := false
	result := &blb.UpdateHTTPListenerArgs{
		ClientToken:  buildClientToken(),
		ListenerPort: uint16(d.Get("listener_port").(int)),
		BackendPort:  uint16(d.Get("backend_port").(int)),
		Scheduler:    d.Get("scheduler").(string),
	}

	update = d.HasChange("scheduler")
	if result.Scheduler != "RoundRobin" && result.Scheduler != "LeastConnection" {
		return false, nil, fmt.Errorf("HTTP Listener scheduler only support [RoundRobin, LeastConnection], but you set: %s", result.Scheduler)
	}
	if v, ok := d.GetOk("health_check_timeout_in_second"); ok {
		if !update {
			result.HealthCheckTimeoutInSecond = v.(int)
		}
	}
	if v, ok := d.GetOk("health_check_interval"); ok {
		if !update {
			result.HealthCheckInterval = v.(int)
		}
	}
	if v, ok := d.GetOk("unhealthy_threshold"); ok {
		if !update {
			result.UnhealthyThreshold = v.(int)
		}
	}
	if v, ok := d.GetOk("healthy_threshold"); ok {
		if !update {
			result.HealthyThreshold = v.(int)
		}
	}
	if v, ok := d.GetOk("health_check_uri"); ok {
		if !update {
			result.HealthCheckURI = v.(string)
		}
	}
	if v, ok := d.GetOk("health_check_port"); ok {
		if !update {
			result.HealthCheckPort = uint16(v.(int))
		}
	}
	if v, ok := d.GetOk("health_check_normal_status"); ok {
		if !update {
			result.HealthCheckNormalStatus = v.(string)
		}
	}
	if v, ok := d.GetOk("keep_session"); ok {
		if !update {
			update = d.HasChange("keep_session")
		}

		result.KeepSession = v.(bool)
	}

	if v, ok := d.GetOk("keep_session_type"); ok {
		if !update {
			update = d.HasChange("keep_session_type")
		}

		result.KeepSessionType = v.(string)
	}

	if v, ok := d.GetOk("keep_session_timeout"); ok {
		if !update {
			update = d.HasChange("keep_session_timeout")
		}

		result.KeepSessionDuration = v.(int)
	}

	if v, ok := d.GetOk("keep_session_cookie_name"); ok {
		if !update {
			update = d.HasChange("keep_session_cookie_name")
		}

		result.KeepSessionCookieName = v.(string)
	}

	if v, ok := d.GetOk("x_forwarded_for"); ok {
		if !update {
			update = d.HasChange("x_forwarded_for")
		}

		result.XForwardedFor = v.(bool)
	}

	if v, ok := d.GetOk("server_timeout"); ok {
		if !update {
			update = d.HasChange("server_timeout")
		}

		result.ServerTimeout = v.(int)
	}

	if v, ok := d.GetOk("redirect_port"); ok {
		if !update {
			update = d.HasChange("redirect_port")
		}

		result.RedirectPort = uint16(v.(int))
	}

	return update, result, nil
}

func buildBaiduCloudUpdateBLBHTTPSListenerArgs(d *schema.ResourceData, meta interface{}) (bool, *blb.UpdateHTTPSListenerArgs, error) {
	update := false
	result := &blb.UpdateHTTPSListenerArgs{
		ClientToken:  buildClientToken(),
		ListenerPort: uint16(d.Get("listener_port").(int)),
		Scheduler:    d.Get("scheduler").(string),
	}

	update = d.HasChange("scheduler")
	if result.Scheduler != "RoundRobin" && result.Scheduler != "LeastConnection" {
		return false, nil, fmt.Errorf("HTTPS Listener scheduler only support [RoundRobin, LeastConnection], but you set: %s", result.Scheduler)
	}

	if v, ok := d.GetOk("health_check_timeout_in_second"); ok {
		if !update {
			result.HealthCheckTimeoutInSecond = v.(int)
		}
	}
	if v, ok := d.GetOk("health_check_interval"); ok {
		if !update {
			result.HealthCheckInterval = v.(int)
		}
	}
	if v, ok := d.GetOk("unhealthy_threshold"); ok {
		if !update {
			result.UnhealthyThreshold = v.(int)
		}
	}
	if v, ok := d.GetOk("healthy_threshold"); ok {
		if !update {
			result.HealthyThreshold = v.(int)
		}
	}
	if v, ok := d.GetOk("health_check_uri"); ok {
		if !update {
			result.HealthCheckURI = v.(string)
		}
	}
	if v, ok := d.GetOk("health_check_port"); ok {
		if !update {
			result.HealthCheckPort = uint16(v.(int))
		}
	}
	if v, ok := d.GetOk("health_check_normal_status"); ok {
		if !update {
			result.HealthCheckNormalStatus = v.(string)
		}
	}

	if v, ok := d.GetOk("keep_session"); ok {
		if !update {
			update = d.HasChange("keep_session")
		}

		result.KeepSession = v.(bool)
	}

	if v, ok := d.GetOk("keep_session_type"); ok {
		if !update {
			update = d.HasChange("keep_session_type")
		}

		result.KeepSessionType = v.(string)
	}

	if v, ok := d.GetOk("keep_session_timeout"); ok {
		if !update {
			update = d.HasChange("keep_session_timeout")
		}

		result.KeepSessionDuration = v.(int)
	}

	if v, ok := d.GetOk("keep_session_cookie_name"); ok {
		if !update {
			update = d.HasChange("keep_session_cookie_name")
		}

		result.KeepSessionCookieName = v.(string)
	}

	if v, ok := d.GetOk("x_forwarded_for"); ok {
		if !update {
			update = d.HasChange("x_forwarded_for")
		}

		result.XForwardedFor = v.(bool)
	}

	if v, ok := d.GetOk("server_timeout"); ok {
		if !update {
			update = d.HasChange("server_timeout")
		}

		result.ServerTimeout = v.(int)
	}

	if v, ok := d.GetOk("cert_ids"); ok {
		if !update {
			update = d.HasChange("cert_ids")
		}
		for _, id := range v.(*schema.Set).List() {
			result.CertIds = append(result.CertIds, id.(string))
		}
	}
	if len(result.CertIds) <= 0 {
		return false, nil, fmt.Errorf("HTTPS Listener require cert, but not set")
	}

	//if v, ok := d.GetOk("dual_auth"); ok {
	//	if !update {
	//		update = d.HasChange("dual_auth")
	//	}

	//result.DualAuth = v.(bool)
	//}

	if v, ok := d.GetOk("client_cert_ids"); ok {
		if !update {
			update = d.HasChange("client_cert_ids")
		}

		for _, id := range v.(*schema.Set).List() {
			result.CertIds = append(result.CertIds, id.(string))
		}
	}

	return update, result, nil
}

func buildBaiduCloudUpdateBLBSSLListenerArgs(d *schema.ResourceData, meta interface{}) (bool, *blb.UpdateSSLListenerArgs, error) {
	update := false
	result := &blb.UpdateSSLListenerArgs{
		ClientToken:  buildClientToken(),
		ListenerPort: uint16(d.Get("listener_port").(int)),
		Scheduler:    d.Get("scheduler").(string),
	}

	update = d.HasChange("scheduler")
	if v, ok := d.GetOk("health_check_timeout_in_second"); ok {
		if !update {
			result.HealthCheckTimeoutInSecond = v.(int)
		}
	}
	if v, ok := d.GetOk("health_check_interval"); ok {
		if !update {
			result.HealthCheckInterval = v.(int)
		}
	}
	if v, ok := d.GetOk("unhealthy_threshold"); ok {
		if !update {
			result.UnhealthyThreshold = v.(int)
		}
	}
	if v, ok := d.GetOk("healthy_threshold"); ok {
		if !update {
			result.HealthyThreshold = v.(int)
		}
	}
	if v, ok := d.GetOk("cert_ids"); ok {
		if !update {
			update = d.HasChange("cert_ids")
		}

		for _, id := range v.(*schema.Set).List() {
			result.CertIds = append(result.CertIds, id.(string))
		}
	}
	if len(result.CertIds) <= 0 {
		return false, nil, fmt.Errorf("SSL Listener require cert, but not set")
	}

	if v, ok := d.GetOk("encryption_type"); ok {
		if !update {
			update = d.HasChange("encryption_type")
		}

		result.EncryptionType = v.(string)
	}

	if v, ok := d.GetOk("encryption_protocols"); ok {
		if !update {
			update = d.HasChange("encryption_protocols")
		}

		for _, p := range v.(*schema.Set).List() {
			result.EncryptionProtocols = append(result.EncryptionProtocols, p.(string))
		}
	}

	if v, ok := d.GetOk("dual_auth"); ok {
		if !update {
			update = d.HasChange("dual_auth")
		}

		result.DualAuth = v.(bool)
	}

	if v, ok := d.GetOk("client_cert_ids"); ok {
		if !update {
			update = d.HasChange("client_cert_ids")
		}

		for _, id := range v.(*schema.Set).List() {
			result.ClientCertIds = append(result.ClientCertIds, id.(string))
		}
	}

	return update, result, nil
}
