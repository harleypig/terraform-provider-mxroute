package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccCheckSpamWhitelistEntryDestroy confirms the entry is gone from the
// domain's spam whitelist after the test.
func testAccCheckSpamWhitelistEntryDestroy(t *testing.T, domain, entry string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		return checkGoneInList(t, "/domains/"+domain+"/spam/whitelist", fmt.Sprintf("spam whitelist entry %q on domain %q", entry, domain), func(e *string) bool { return *e == entry })
	}
}

func TestAccSpamWhitelistEntryResource(t *testing.T) {
	domain := testAccTestDomain(t)
	entry := "*@trusted.example"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSpamWhitelistEntryDestroy(t, domain, entry),
		Steps: []resource.TestStep{
			{
				Config: testAccSpamWhitelistEntryResourceConfig(domain, entry),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_spam_whitelist_entry.test", "domain", domain),
					resource.TestCheckResourceAttr("mxroute_spam_whitelist_entry.test", "entry", entry),
					resource.TestCheckResourceAttr("mxroute_spam_whitelist_entry.test", "id", domain+"/"+entry),
				),
			},
			{
				ResourceName:      "mxroute_spam_whitelist_entry.test",
				ImportState:       true,
				ImportStateId:     domain + "/" + entry,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccSpamWhitelistEntryResourceConfig(domain, entry string) string {
	return fmt.Sprintf(`
resource "mxroute_domain" "test" {
  domain = %[1]q
}

resource "mxroute_spam_whitelist_entry" "test" {
  domain = mxroute_domain.test.domain
  entry  = %[2]q
}
`, domain, entry)
}
