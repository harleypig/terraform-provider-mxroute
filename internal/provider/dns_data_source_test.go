package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDNSDataSource(t *testing.T) {
	domain := testAccTestDomain(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDomainDestroy(t, domain),
		Steps: []resource.TestStep{
			{
				// Create the throwaway domain, then read its DNS records
				// through the data source in the same config so the test is
				// self-contained.
				Config: testAccDNSDataSourceConfig(domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.mxroute_dns.test", "domain", domain),
					resource.TestCheckResourceAttr("data.mxroute_dns.test", "id", domain),
					resource.TestCheckResourceAttrSet("data.mxroute_dns.test", "mx_records.#"),
				),
			},
		},
	})
}

func testAccDNSDataSourceConfig(domain string) string {
	return fmt.Sprintf(`
resource "mxroute_domain" "test" {
  domain = %[1]q
}

data "mxroute_dns" "test" {
  domain = mxroute_domain.test.domain
}
`, domain)
}
