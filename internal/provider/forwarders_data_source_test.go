package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccForwardersDataSource(t *testing.T) {
	domain := testAccTestDomain(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDomainDestroy(t, domain),
		Steps: []resource.TestStep{
			{
				// Create a throwaway domain with one forwarder, then list the
				// domain's forwarders and assert the element content — not just
				// the count. With exactly one forwarder, forwarders.0 is
				// deterministic, so alias/email/destinations must round-trip
				// through the list decode.
				Config: fmt.Sprintf(`
resource "mxroute_domain" "test" {
  domain = %[1]q
}

resource "mxroute_forwarder" "test" {
  domain       = mxroute_domain.test.domain
  alias        = "sales"
  destinations = ["owner@example.net"]
}

data "mxroute_forwarders" "test" {
  domain     = mxroute_domain.test.domain
  depends_on = [mxroute_forwarder.test]
}
`, domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.mxroute_forwarders.test", "domain", domain),
					resource.TestCheckResourceAttr("data.mxroute_forwarders.test", "id", domain),
					resource.TestCheckResourceAttr("data.mxroute_forwarders.test", "forwarders.#", "1"),
					resource.TestCheckResourceAttr("data.mxroute_forwarders.test", "forwarders.0.alias", "sales"),
					resource.TestCheckResourceAttr("data.mxroute_forwarders.test", "forwarders.0.email", "sales@"+domain),
					resource.TestCheckResourceAttr("data.mxroute_forwarders.test", "forwarders.0.destinations.#", "1"),
					resource.TestCheckResourceAttr("data.mxroute_forwarders.test", "forwarders.0.destinations.0", "owner@example.net"),
				),
			},
		},
	})
}
