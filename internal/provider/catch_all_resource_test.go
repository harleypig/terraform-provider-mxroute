package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestCatchAllAddressSet(t *testing.T) {
	tests := []struct {
		name string
		addr types.String
		want bool
	}{
		{"null is unset", types.StringNull(), false},
		{"unknown is unset", types.StringUnknown(), false},
		{"empty string is unset", types.StringValue(""), false},
		{"non-empty is set", types.StringValue("harleypig@harleypig.com"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := catchAllAddressSet(tt.addr); got != tt.want {
				t.Errorf("catchAllAddressSet(%v) = %v, want %v", tt.addr, got, tt.want)
			}
		})
	}
}

// testAccCheckCatchAllDestroy confirms the catch-all policy is back at its
// "fail" default after the test — or that the parent domain is gone, which
// also leaves no policy behind.
func testAccCheckCatchAllDestroy(t *testing.T, domain string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := NewClient(ClientConfig{
			Server:   os.Getenv("MXROUTE_SERVER"),
			Username: os.Getenv("MXROUTE_USERNAME"),
			APIKey:   os.Getenv("MXROUTE_API_KEY"),
		})

		var api CatchAll

		err := client.Do(t.Context(), "GET", catchAllPath(domain), nil, &api)
		if err != nil {
			if IsNotFound(err) {
				return nil
			}

			return fmt.Errorf("checking catch-all destroy: %w", err)
		}

		if api.Type != "fail" {
			return fmt.Errorf("catch-all policy for %q is %q after destroy, want \"fail\"", domain, api.Type)
		}

		return nil
	}
}

func TestAccCatchAllResource(t *testing.T) {
	domain := testAccTestDomain(t)
	address := "postmaster@" + domain

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCatchAllDestroy(t, domain),
		Steps: []resource.TestStep{
			{
				Config: testAccCatchAllResourceConfigBlackhole(domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_catch_all.test", "domain", domain),
					resource.TestCheckResourceAttr("mxroute_catch_all.test", "type", "blackhole"),
					resource.TestCheckResourceAttr("mxroute_catch_all.test", "id", domain),
				),
			},
			{
				Config: testAccCatchAllResourceConfigAddress(domain, address),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_catch_all.test", "type", "address"),
					resource.TestCheckResourceAttr("mxroute_catch_all.test", "address", address),
				),
			},
			{
				ResourceName:      "mxroute_catch_all.test",
				ImportState:       true,
				ImportStateId:     domain,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCatchAllResourceConfigBlackhole(domain string) string {
	return fmt.Sprintf(`
resource "mxroute_domain" "test" {
  domain = %[1]q
}

resource "mxroute_catch_all" "test" {
  domain = mxroute_domain.test.domain
  type   = "blackhole"
}
`, domain)
}

func testAccCatchAllResourceConfigAddress(domain, address string) string {
	return fmt.Sprintf(`
resource "mxroute_domain" "test" {
  domain = %[1]q
}

resource "mxroute_catch_all" "test" {
  domain  = mxroute_domain.test.domain
  type    = "address"
  address = %[2]q
}
`, domain, address)
}
