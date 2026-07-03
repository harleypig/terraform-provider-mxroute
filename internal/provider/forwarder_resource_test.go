package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccCheckForwarderDestroy confirms the forwarder is gone after the test.
func testAccCheckForwarderDestroy(t *testing.T, domain, alias string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := NewClient(ClientConfig{
			Server:   os.Getenv("MXROUTE_SERVER"),
			Username: os.Getenv("MXROUTE_USERNAME"),
			APIKey:   os.Getenv("MXROUTE_API_KEY"),
		})

		var forwarders []forwarderAPIModel

		err := client.Do(t.Context(), "GET", "/domains/"+domain+"/forwarders", nil, &forwarders)
		if err != nil {
			// The parent domain being gone also means the forwarder is gone.
			if IsNotFound(err) {
				return nil
			}

			return fmt.Errorf("checking forwarder destroy: %w", err)
		}

		for i := range forwarders {
			if forwarders[i].Alias == alias {
				return fmt.Errorf("forwarder %q on domain %q still exists after destroy", alias, domain)
			}
		}

		return nil
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
					resource.TestCheckResourceAttr("mxroute_forwarder.test", "destinations.0", "owner@example.net"),
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
