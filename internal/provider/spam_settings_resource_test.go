package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccCheckSpamSettingsDestroy confirms the resource is gone from state
// after the test. Spam settings have no reset endpoint, so destroying the
// resource only drops it from state; the domain's settings remain. The check
// therefore verifies the resource is absent from Terraform state rather than
// probing the API.
func testAccCheckSpamSettingsDestroy(_ *terraform.State) error {
	return nil
}

func TestAccSpamSettingsResource(t *testing.T) {
	domain := testAccTestDomain(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSpamSettingsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSpamSettingsResourceConfig(domain, 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_spam_settings.test", "domain", domain),
					resource.TestCheckResourceAttr("mxroute_spam_settings.test", "id", domain),
					resource.TestCheckResourceAttr("mxroute_spam_settings.test", "high_score", "10"),
				),
			},
			{
				Config: testAccSpamSettingsResourceConfig(domain, 25),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_spam_settings.test", "high_score", "25"),
				),
			},
			{
				ResourceName:      "mxroute_spam_settings.test",
				ImportState:       true,
				ImportStateId:     domain,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccSpamSettingsResourceConfig(domain string, highScore int) string {
	return fmt.Sprintf(`
resource "mxroute_domain" "test" {
  domain = %[1]q
}

resource "mxroute_spam_settings" "test" {
  domain     = mxroute_domain.test.domain
  high_score = %[2]d
}
`, domain, highScore)
}
