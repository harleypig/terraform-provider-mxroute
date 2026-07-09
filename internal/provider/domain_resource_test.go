package provider

import (
	"fmt"
	"os"
	"regexp"
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

// testAccUnverifiedDomain returns a domain that is NOT ownership-verified on the
// account, for the negative "verification required" test. It is a separate knob
// from testAccTestDomain because that one is verified (adds succeed): here we
// need one that is deliberately left unverified (harleydev.com, per harleydev's
// e2e/mxroute.md). Skips when unset, and never the live domain.
func testAccUnverifiedDomain(t *testing.T) string {
	t.Helper()

	domain := os.Getenv("MXROUTE_TEST_UNVERIFIED_DOMAIN")
	if domain == "" {
		t.Skip("MXROUTE_TEST_UNVERIFIED_DOMAIN not set; skipping unverified-domain 422 test")
	}

	if domain == "harleypig.com" {
		t.Fatal("MXROUTE_TEST_UNVERIFIED_DOMAIN must be a throwaway domain, never the live harleypig.com")
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

// TestAccDomainResource_unverified422 asserts that adding an UNVERIFIED domain
// fails with MXroute's HTTP 422 "Domain verification required". This is the
// negative path native `terraform test` can't express — its expect_failures
// catches only condition/validation failures, not a provider apply-error — so
// it lives here as scenario 6 of harleydev's e2e/mxroute.md, rerouted to Go.
func TestAccDomainResource_unverified422(t *testing.T) {
	domain := testAccUnverifiedDomain(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDomainDestroy(t, domain),
		Steps: []resource.TestStep{
			{
				Config:      testAccDomainResourceConfig(domain),
				ExpectError: regexp.MustCompile(`(?i)verification required`),
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
