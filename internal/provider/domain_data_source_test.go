package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDomainDataSource(t *testing.T) {
	domain := testAccTestDomain(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDomainDestroy(t, domain),
		Steps: []resource.TestStep{
			{
				// Create the throwaway domain, then read it back through the
				// data source in the same config so the test is self-contained.
				Config: testAccDomainDataSourceConfig(domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.mxroute_domain.test", "domain", domain),
					resource.TestCheckResourceAttr("data.mxroute_domain.test", "id", domain),
					resource.TestCheckResourceAttrSet("data.mxroute_domain.test", "mail_hosting"),
				),
			},
		},
	})
}

func testAccDomainDataSourceConfig(domain string) string {
	return fmt.Sprintf(`
resource "mxroute_domain" "test" {
  domain = %[1]q
}

data "mxroute_domain" "test" {
  domain = mxroute_domain.test.domain
}
`, domain)
}
