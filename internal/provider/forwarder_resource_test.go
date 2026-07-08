package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccCheckForwarderDestroy confirms the forwarder is gone after the test.
func testAccCheckForwarderDestroy(t *testing.T, domain, alias string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		return checkGoneInList(t, "/domains/"+domain+"/forwarders", fmt.Sprintf("forwarder %q on domain %q", alias, domain), func(f *Forwarder) bool { return f.Alias == alias })
	}
}

func TestAccForwarderResource(t *testing.T) {
	domain := testAccTestDomain(t)
	alias := "sales"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckForwarderDestroy(t, domain, alias),
		Steps: []resource.TestStep{
			{
				Config: testAccForwarderResourceConfig(domain, alias),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_forwarder.test", "domain", domain),
					resource.TestCheckResourceAttr("mxroute_forwarder.test", "alias", alias),
					resource.TestCheckResourceAttr("mxroute_forwarder.test", "id", domain+"/"+alias),
					resource.TestCheckResourceAttr("mxroute_forwarder.test", "destinations.#", "1"),
					// destinations is a Set — elements are hashed, not indexed.
					resource.TestCheckTypeSetElemAttr("mxroute_forwarder.test", "destinations.*", "owner@example.net"),
					resource.TestCheckResourceAttrSet("mxroute_forwarder.test", "email"),
				),
			},
			{
				ResourceName:      "mxroute_forwarder.test",
				ImportState:       true,
				ImportStateId:     domain + "/" + alias,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccForwarderResource_plusInAlias exercises pathSeg's encoding of a `+`
// in a path segment. The forwarder is created via the request body, but its
// teardown DELETE targets /domains/{domain}/forwarders/{alias} with
// pathSeg(alias) — CheckDestroy fails if the `+` alias isn't matched (the
// forwarder lingers), which is exactly the "does the API need @/+
// percent-encoded" question. If this fails live, switch pathSeg to a stricter
// encoder that escapes `@`/`+` too.
func TestAccForwarderResource_plusInAlias(t *testing.T) {
	domain := testAccTestDomain(t)
	alias := "foo+bar"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckForwarderDestroy(t, domain, alias),
		Steps: []resource.TestStep{
			{
				Config: testAccForwarderResourceConfig(domain, alias),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_forwarder.test", "alias", alias),
					resource.TestCheckResourceAttr("mxroute_forwarder.test", "id", domain+"/"+alias),
					// Read filters the list by alias, so a passing read already
					// proves the `+` survives create + list; CheckDestroy proves
					// the DELETE path segment matches it.
					resource.TestCheckResourceAttr("mxroute_forwarder.test", "destinations.#", "1"),
				),
			},
		},
	})
}

func testAccForwarderResourceConfig(domain, alias string) string {
	return fmt.Sprintf(`
resource "mxroute_domain" "test" {
  domain = %[1]q
}

resource "mxroute_forwarder" "test" {
  domain       = mxroute_domain.test.domain
  alias        = %[2]q
  destinations = ["owner@example.net"]
}
`, domain, alias)
}
