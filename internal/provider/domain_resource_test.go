package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccTestDomain returns the throwaway domain acceptance tests may create
// and destroy. It must be set and must never be the live production domain.
func testAccTestDomain(t *testing.T) string {
	t.Helper()

	domain := os.Getenv("MXROUTE_TEST_DOMAIN")
	if domain == "" {
		t.Skip("MXROUTE_TEST_DOMAIN not set; skipping domain acceptance test")
	}

	if domain == "harleypig.com" {
		t.Fatal("MXROUTE_TEST_DOMAIN must be a throwaway domain, never the live harleypig.com")
	}

	return domain
}

// testAccCheckDomainDestroy confirms the domain is gone after the test.
func testAccCheckDomainDestroy(t *testing.T, domain string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		return checkGoneSingle[Domain](t, "/domains/"+domain, fmt.Sprintf("domain %q", domain))
	}
}

func TestAccDomainResource(t *testing.T) {
	domain := testAccTestDomain(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDomainDestroy(t, domain),
		Steps: []resource.TestStep{
			{
				Config: testAccDomainResourceConfig(domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_domain.test", "domain", domain),
					resource.TestCheckResourceAttr("mxroute_domain.test", "id", domain),
					resource.TestCheckResourceAttrSet("mxroute_domain.test", "mail_hosting"),
				),
			},
			{
				Config: testAccDomainResourceConfigMailHosting(domain, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_domain.test", "domain", domain),
					resource.TestCheckResourceAttr("mxroute_domain.test", "mail_hosting", "false"),
				),
			},
			{
				ResourceName:      "mxroute_domain.test",
				ImportState:       true,
				ImportStateId:     domain,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccDomainResourceConfig(domain string) string {
	return fmt.Sprintf(`
resource "mxroute_domain" "test" {
  domain = %[1]q
}
`, domain)
}

func testAccDomainResourceConfigMailHosting(domain string, mailHosting bool) string {
	return fmt.Sprintf(`
resource "mxroute_domain" "test" {
  domain       = %[1]q
  mail_hosting = %[2]t
}
`, domain, mailHosting)
}
