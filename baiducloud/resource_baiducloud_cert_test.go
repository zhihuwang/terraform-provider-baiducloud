package baiducloud

import (
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/baidubce/bce-sdk-go/services/cert"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

	"github.com/terraform-providers/terraform-provider-baiducloud/baiducloud/connectivity"
)

const (
	testAccCertResourceType = "baiducloud_cert"
	testAccCertResourceName = testAccCertResourceType + "." + BaiduCloudTestResourceName
)

func init() {
	resource.AddTestSweepers(testAccCertResourceType, &resource.Sweeper{
		Name: testAccCertResourceType,
		F:    testSweepCerts,
	})
}

func testSweepCerts(region string) error {
	rawClient, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("get BaiduCloud client error: %s", err)
	}
	client := rawClient.(*connectivity.BaiduClient)

	raw, err := client.WithCertClient(func(client *cert.Client) (i interface{}, e error) {
		return client.ListCerts()
	})
	if err != nil {
		return fmt.Errorf("get Certs error: %s", err)
	}

	for _, c := range raw.(*cert.ListCertResult).Certs {
		if !strings.HasPrefix(c.CertName, BaiduCloudTestResourceTypeName) {
			log.Printf("[INFO] Skipping Cert: %s (%s)", c.CertName, c.CertId)
			continue
		}

		log.Printf("[INFO] Deleting Cert: %s (%s)", c.CertName, c.CertId)

		_, err := client.WithCertClient(func(client *cert.Client) (i interface{}, e error) {
			return nil, client.DeleteCert(c.CertId)
		})
		if err != nil {
			log.Printf("[ERROR] Failed to delete Cert %s (%s)", c.CertName, c.CertId)
		}
	}

	return nil
}

//lintignore:AT003
func TestAccBaiduCloudCert(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCertDestory,

		Steps: []resource.TestStep{
			{
				Config: testAccCertConfig(BaiduCloudTestResourceTypeNameCert),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaiduCloudDataSourceId(testAccCertResourceName),
					resource.TestCheckResourceAttr(testAccCertResourceName, "cert_name", BaiduCloudTestResourceTypeNameCert),
					resource.TestCheckResourceAttr(testAccCertResourceName, "cert_type", "1"),
				),
			},
			{
				Config: testAccCertConfigUpdate(BaiduCloudTestResourceTypeNameCert),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBaiduCloudDataSourceId(testAccCertResourceName),
					resource.TestCheckResourceAttr(testAccCertResourceName, "cert_name", BaiduCloudTestResourceTypeNameCert+"-update"),
					resource.TestCheckResourceAttr(testAccCertResourceName, "cert_type", "1"),
				),
			},
		},
	})
}

func testAccCertDestory(s *terraform.State) error {
	client := testAccProvider.Meta().(*connectivity.BaiduClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != testAccCertResourceType {
			continue
		}

		_, err := client.WithCertClient(func(client *cert.Client) (i interface{}, e error) {
			return client.GetCertMeta(rs.Primary.ID)
		})
		if err != nil {
			if NotFoundError(err) || IsExceptedErrors(err, []string{"not exist"}) {
				continue
			}
			return WrapError(err)
		}
		return WrapError(Error("Cert still exist"))
	}

	return nil
}

func testAccCertConfig(name string) string {
	return fmt.Sprintf(`
resource "baiducloud_cert" "default" {
  cert_name         = "%s"
  cert_server_data  = "-----BEGIN CERTIFICATE-----\n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\n-----END CERTIFICATE-----\n-----BEGIN CERTIFICATE-----\nMIIErjCCA5agAwIBAgIQBYAmfwbylVM0jhwYWl7uLjANBgkqhkiG9w0BAQsFADBhMQswCQYDVQQGEwJVUzEVMBMGA1UEChMMRGlnaUNlcnQgSW5jMRkwFwYDVQQLExB3d3cuZGlnaWNlcnQuY29tMSAwHgYDVQQDExdEaWdpQ2VydCBHbG9iYWwgUm9vdCBDQTAeFw0xNzEyMDgxMjI4MjZaFw0yNzEyMDgxMjI4MjZaMHIxCzAJBgNVBAYTAkNOMSUwIwYDVQQKExxUcnVzdEFzaWEgVGVjaG5vbG9naWVzLCBJbmMuMR0wGwYDVQQLExREb21haW4gVmFsaWRhdGVkIFNTTDEdMBsGA1UEAxMUVHJ1c3RBc2lhIFRMUyBSU0EgQ0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCgWa9X+ph+wAm8Yh1Fk1MjKbQ5QwBOOKVaZR/OfCh+F6f93u7vZHGcUU/lvVGgUQnbzJhR1UV2epJae+m7cxnXIKdD0/VS9btAgwJszGFvwoqXeaCqFoP71wPmXjjUwLT70+qvX4hdyYfOJcjeTz5QKtg8zQwxaK9x4JT9CoOmoVdVhEBAiD3DwR5fFgOHDwwGxdJWVBvktnoAzjdTLXDdbSVC5jZ0u8oq9BiTDv7jAlsB5F8aZgvSZDOQeFrwaOTbKWSEInEhnchKZTD1dz6aBlk1xGEI5PZWAnVAba/ofH33ktymaTDsE6xRDnW97pDkimCRak6CEbfe3dXw6OV5AgMBAAGjggFPMIIBSzAdBgNVHQ4EFgQUf9OZ86BHDjEAVlYijrfMnt3KAYowHwYDVR0jBBgwFoAUA95QNVbRTLtm8KPiGxvDl7I90VUwDgYDVR0PAQH/BAQDAgGGMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjASBgNVHRMBAf8ECDAGAQH/AgEAMDQGCCsGAQUFBwEBBCgwJjAkBggrBgEFBQcwAYYYaHR0cDovL29jc3AuZGlnaWNlcnQuY29tMEIGA1UdHwQ7MDkwN6A1oDOGMWh0dHA6Ly9jcmwzLmRpZ2ljZXJ0LmNvbS9EaWdpQ2VydEdsb2JhbFJvb3RDQS5jcmwwTAYDVR0gBEUwQzA3BglghkgBhv1sAQIwKjAoBggrBgEFBQcCARYcaHR0cHM6Ly93d3cuZGlnaWNlcnQuY29tL0NQUzAIBgZngQwBAgEwDQYJKoZIhvcNAQELBQADggEBAK3dVOj5dlv4MzK2i233lDYvyJ3slFY2X2HKTYGte8nbK6i5/fsDImMYihAkp6VaNY/en8WZ5qcrQPVLuJrJDSXT04NnMeZOQDUoj/NHAmdfCBB/h1bZ5OGK6Sf1h5Yx/5wR4f3TUoPgGlnU7EuPISLNdMRiDrXntcImDAiRvkh5GJuH4YCVE6XEntqaNIgGkRwxKSgnU3Id3iuFbW9FUQ9Qqtb1GX91AJ7i4153TikGgYCdwYkBURD8gSVe8OAco6IfZOYt/TEwii1Ivi1CqnuUlWpsF1LdQNIdfbW3TSe0BhQa7ifbVIfvPWHYOu3rkg1ZeMo6XRU9B4n5VyJYRmE=\n-----END CERTIFICATE-----"
  cert_private_data = "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEAkZCmcbCZyzdYQ379jndWgcKB9Q0/ITUYE2sUlh39HwyAJYP9DP2fStqF9pW+KVYJDPlhdFbYTrUqp+dO9001Hj3bL2JKMwx6zKmKtjUecpSkFtOYKAF+e6odhcS5HUTq0maydjvLSp1kPKd0Q6n4GW14+gkjoUvMVWK59FmjfO47yqETTWK/zA6JAp4ZMXuI6ujFlsnoxY4nFD0+HKiPNkYaFwDr0AUEwdAaDG/fp7/AP2xnQLdGXVn/gflulflUmA9tqwdF2ul9UmsY8iZcbPUbO0EtC78LOGPCgC5SsYVioUCzGO1HCbnDz3jvshAsZ/EMR07KBzqretZwEnuTSQIDAQABAoIBAAzBl4cfWfLljY4TVbFY7ZNJ0i1Wilbkz2XQPJ8aegFGYqp8TROI3EnpKX6I89UCgvYzRSI2rsEC/lMgIZrpa1i+70jRPRMJKm+/VyENjvatO6NRH/ni26HcWrb2HN90Qnx1XyPzrHvZnBxL876EPseCVkIvGoNliulb+/4Y/DXpNthA28UOB9RafPsEoDNinrTqlZf0gNLxm1LOgcj/NEqsDwuwzwfCky9GAhQgZpwic2IAEwKoCbfeRNNraVgG+IdCC8Nn3/uMcy9Zft3fV7xNE6HdfkW1SKnEvN+sFxKhH7ad0FNtaE+kSAcxTWXOg/xErvUBIcDrZv23BgN4JVMCgYEAwiNb00eRuBcPTHAaEb9JqrFRtUlqLnFJe1ang1QRfn+FrlTnijGACTjEFpzaXavaGNKi+To8OZjSTL2OW6ewEwSA9siPXUkq3ldPj5uPIhr80Jn1Ox/K5+X5ZBkQg8Iw9GIY6P6Kgf/prihVIbGZVNa0U/8H/1RvQIBxvA21dfMCgYEAv/L8iGiSwcgqMv0NTzfiW4fA9L7yLE04mfs9QI1V/uHPX5ufb/Y3LCS1RSuOdjrCdD2Ru7OKMi1v7mwPg1+NJBZjLIlCw/oVCJZabd8KGXZUNSH+PNuQAbIGdotEpO+LPgVgwi4ovrx6oJYEED/1FFjfU2bBFfuZtrDBWz2yNNMCgYEAvdoKQJHq5RZX9a5jMBvbFLwXZawH1Kcg7ycM5hdejFB1EMkjLTe/OEV1LY/y1EvtGv1SN1xF7SWP81AkWWmhfNeYrr3vxZB6Bbloqs27qeSue+kzssAik6mIu+TvC4rqiPMt3RyfowX7Jj93EV42zoqxCruKvJ17tp5lmzvkyxUCgYBRN60mwqimGd3RKUWCaXD7rZs1c73ghOQYMzgdoi/q4vztxVlW9GUv5nBUzjM/T2mL6alKNJOa26LqzQpbWgjMZjScWY/IgH553bRxnNgXIfxLZxC+C2EJdpxJeHAZIcpW+cuRHhrbacCxRgh+H7HBZEFKdsXoWUcXB/8obhiDRQKBgCwOE+1hfrV7/gFaMBWSML1n+LVV2ns80jCDtkhN9yF+9iJTjMwW4wuvFx8t8o2XICOwJPog4IvXFJLVZeed/zhgqe4qImHRW0aMYGyGEpgkLtHIFFFCxGd57Df/qEbUL55LU53rlCv2QKVBBs/6XDkiVRBk8izT7ihF2U8qb6t4\n-----END RSA PRIVATE KEY-----"
}
`, name)
}

func testAccCertConfigUpdate(name string) string {
	return fmt.Sprintf(`
resource "baiducloud_cert" "default" {
  cert_name         = "%s"
  cert_server_data  = "-----BEGIN CERTIFICATE-----\n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\n-----END CERTIFICATE-----\n-----BEGIN CERTIFICATE-----\nMIIErjCCA5agAwIBAgIQBYAmfwbylVM0jhwYWl7uLjANBgkqhkiG9w0BAQsFADBhMQswCQYDVQQGEwJVUzEVMBMGA1UEChMMRGlnaUNlcnQgSW5jMRkwFwYDVQQLExB3d3cuZGlnaWNlcnQuY29tMSAwHgYDVQQDExdEaWdpQ2VydCBHbG9iYWwgUm9vdCBDQTAeFw0xNzEyMDgxMjI4MjZaFw0yNzEyMDgxMjI4MjZaMHIxCzAJBgNVBAYTAkNOMSUwIwYDVQQKExxUcnVzdEFzaWEgVGVjaG5vbG9naWVzLCBJbmMuMR0wGwYDVQQLExREb21haW4gVmFsaWRhdGVkIFNTTDEdMBsGA1UEAxMUVHJ1c3RBc2lhIFRMUyBSU0EgQ0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCgWa9X+ph+wAm8Yh1Fk1MjKbQ5QwBOOKVaZR/OfCh+F6f93u7vZHGcUU/lvVGgUQnbzJhR1UV2epJae+m7cxnXIKdD0/VS9btAgwJszGFvwoqXeaCqFoP71wPmXjjUwLT70+qvX4hdyYfOJcjeTz5QKtg8zQwxaK9x4JT9CoOmoVdVhEBAiD3DwR5fFgOHDwwGxdJWVBvktnoAzjdTLXDdbSVC5jZ0u8oq9BiTDv7jAlsB5F8aZgvSZDOQeFrwaOTbKWSEInEhnchKZTD1dz6aBlk1xGEI5PZWAnVAba/ofH33ktymaTDsE6xRDnW97pDkimCRak6CEbfe3dXw6OV5AgMBAAGjggFPMIIBSzAdBgNVHQ4EFgQUf9OZ86BHDjEAVlYijrfMnt3KAYowHwYDVR0jBBgwFoAUA95QNVbRTLtm8KPiGxvDl7I90VUwDgYDVR0PAQH/BAQDAgGGMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjASBgNVHRMBAf8ECDAGAQH/AgEAMDQGCCsGAQUFBwEBBCgwJjAkBggrBgEFBQcwAYYYaHR0cDovL29jc3AuZGlnaWNlcnQuY29tMEIGA1UdHwQ7MDkwN6A1oDOGMWh0dHA6Ly9jcmwzLmRpZ2ljZXJ0LmNvbS9EaWdpQ2VydEdsb2JhbFJvb3RDQS5jcmwwTAYDVR0gBEUwQzA3BglghkgBhv1sAQIwKjAoBggrBgEFBQcCARYcaHR0cHM6Ly93d3cuZGlnaWNlcnQuY29tL0NQUzAIBgZngQwBAgEwDQYJKoZIhvcNAQELBQADggEBAK3dVOj5dlv4MzK2i233lDYvyJ3slFY2X2HKTYGte8nbK6i5/fsDImMYihAkp6VaNY/en8WZ5qcrQPVLuJrJDSXT04NnMeZOQDUoj/NHAmdfCBB/h1bZ5OGK6Sf1h5Yx/5wR4f3TUoPgGlnU7EuPISLNdMRiDrXntcImDAiRvkh5GJuH4YCVE6XEntqaNIgGkRwxKSgnU3Id3iuFbW9FUQ9Qqtb1GX91AJ7i4153TikGgYCdwYkBURD8gSVe8OAco6IfZOYt/TEwii1Ivi1CqnuUlWpsF1LdQNIdfbW3TSe0BhQa7ifbVIfvPWHYOu3rkg1ZeMo6XRU9B4n5VyJYRmE=\n-----END CERTIFICATE-----"
  cert_private_data = "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEAkZCmcbCZyzdYQ379jndWgcKB9Q0/ITUYE2sUlh39HwyAJYP9DP2fStqF9pW+KVYJDPlhdFbYTrUqp+dO9001Hj3bL2JKMwx6zKmKtjUecpSkFtOYKAF+e6odhcS5HUTq0maydjvLSp1kPKd0Q6n4GW14+gkjoUvMVWK59FmjfO47yqETTWK/zA6JAp4ZMXuI6ujFlsnoxY4nFD0+HKiPNkYaFwDr0AUEwdAaDG/fp7/AP2xnQLdGXVn/gflulflUmA9tqwdF2ul9UmsY8iZcbPUbO0EtC78LOGPCgC5SsYVioUCzGO1HCbnDz3jvshAsZ/EMR07KBzqretZwEnuTSQIDAQABAoIBAAzBl4cfWfLljY4TVbFY7ZNJ0i1Wilbkz2XQPJ8aegFGYqp8TROI3EnpKX6I89UCgvYzRSI2rsEC/lMgIZrpa1i+70jRPRMJKm+/VyENjvatO6NRH/ni26HcWrb2HN90Qnx1XyPzrHvZnBxL876EPseCVkIvGoNliulb+/4Y/DXpNthA28UOB9RafPsEoDNinrTqlZf0gNLxm1LOgcj/NEqsDwuwzwfCky9GAhQgZpwic2IAEwKoCbfeRNNraVgG+IdCC8Nn3/uMcy9Zft3fV7xNE6HdfkW1SKnEvN+sFxKhH7ad0FNtaE+kSAcxTWXOg/xErvUBIcDrZv23BgN4JVMCgYEAwiNb00eRuBcPTHAaEb9JqrFRtUlqLnFJe1ang1QRfn+FrlTnijGACTjEFpzaXavaGNKi+To8OZjSTL2OW6ewEwSA9siPXUkq3ldPj5uPIhr80Jn1Ox/K5+X5ZBkQg8Iw9GIY6P6Kgf/prihVIbGZVNa0U/8H/1RvQIBxvA21dfMCgYEAv/L8iGiSwcgqMv0NTzfiW4fA9L7yLE04mfs9QI1V/uHPX5ufb/Y3LCS1RSuOdjrCdD2Ru7OKMi1v7mwPg1+NJBZjLIlCw/oVCJZabd8KGXZUNSH+PNuQAbIGdotEpO+LPgVgwi4ovrx6oJYEED/1FFjfU2bBFfuZtrDBWz2yNNMCgYEAvdoKQJHq5RZX9a5jMBvbFLwXZawH1Kcg7ycM5hdejFB1EMkjLTe/OEV1LY/y1EvtGv1SN1xF7SWP81AkWWmhfNeYrr3vxZB6Bbloqs27qeSue+kzssAik6mIu+TvC4rqiPMt3RyfowX7Jj93EV42zoqxCruKvJ17tp5lmzvkyxUCgYBRN60mwqimGd3RKUWCaXD7rZs1c73ghOQYMzgdoi/q4vztxVlW9GUv5nBUzjM/T2mL6alKNJOa26LqzQpbWgjMZjScWY/IgH553bRxnNgXIfxLZxC+C2EJdpxJeHAZIcpW+cuRHhrbacCxRgh+H7HBZEFKdsXoWUcXB/8obhiDRQKBgCwOE+1hfrV7/gFaMBWSML1n+LVV2ns80jCDtkhN9yF+9iJTjMwW4wuvFx8t8o2XICOwJPog4IvXFJLVZeed/zhgqe4qImHRW0aMYGyGEpgkLtHIFFFCxGd57Df/qEbUL55LU53rlCv2QKVBBs/6XDkiVRBk8izT7ihF2U8qb6t4\n-----END RSA PRIVATE KEY-----"
}
`, name+"-update")
}
