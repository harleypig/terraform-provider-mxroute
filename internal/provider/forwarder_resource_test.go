package provider

import (
	"fmt"
	"regexp"
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
				// Change destinations. Every forwarder attribute is
				// RequiresReplace, so this destroys and recreates the forwarder
				// rather than updating in place; the new destination must
				// round-trip through the replace.
				Config: testAccForwarderResourceConfigDest(domain, alias, "newowner@example.net"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_forwarder.test", "destinations.#", "1"),
					resource.TestCheckTypeSetElemAttr("mxroute_forwarder.test", "destinations.*", "newowner@example.net"),
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

// TestAccForwarderResource_plusInAlias asserts the plan-time alias validator
// rejects a `+` (an out-of-charset character). The MXroute API enforces this
// alias charset server-side — an alias with `+` fails create with HTTP 400
// VALIDATION_ERROR — but does not declare it in its OpenAPI spec, so the
// provider mirrors the rule at plan time (see forwarderAliasPattern). No
// PreCheck and no live account: the validator fires during plan, before any
// API call, so this runs in the default CI gate.
func TestAccForwarderResource_plusInAlias(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "mxroute_forwarder" "test" {
  domain       = "example.com"
  alias        = "foo+bar"
  destinations = ["owner@example.net"]
}`,
				ExpectError: regexp.MustCompile("must start with a letter or number"),
			},
		},
	})
}

func testAccForwarderResourceConfig(domain, alias string) string {
	return testAccForwarderResourceConfigDest(domain, alias, "owner@example.net")
}

// testAccForwarderResourceConfigDest is testAccForwarderResourceConfig with a
// caller-chosen destination, so a second step can change destinations and
// exercise the RequiresReplace path.
func testAccForwarderResourceConfigDest(domain, alias, destination string) string {
	return fmt.Sprintf(`
resource "mxroute_domain" "test" {
  domain = %[1]q
}

resource "mxroute_forwarder" "test" {
  domain       = mxroute_domain.test.domain
  alias        = %[2]q
  destinations = [%[3]q]
}
`, domain, alias, destination)
}
