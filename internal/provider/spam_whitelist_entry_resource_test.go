package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccCheckSpamWhitelistEntryDestroy confirms the entry is gone from the
// domain's spam whitelist after the test.
func testAccCheckSpamWhitelistEntryDestroy(t *testing.T, domain, entry string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := NewClient(ClientConfig{
			Server:   os.Getenv("MXROUTE_SERVER"),
			Username: os.Getenv("MXROUTE_USERNAME"),
			APIKey:   os.Getenv("MXROUTE_API_KEY"),
		})

		var whitelist []string

		err := client.Do(t.Context(), "GET", "/domains/"+domain+"/spam/whitelist", nil, &whitelist)
		if err != nil {
			// The parent domain is destroyed by its own resource, so a missing
			// domain means the entry is gone too.
			if IsNotFound(err) {
				return nil
			}

			return fmt.Errorf("checking spam whitelist entry destroy: %w", err)
		}

		for _, candidate := range whitelist {
			if candidate == entry {
				return fmt.Errorf("spam whitelist entry %q still exists after destroy", entry)
			}
		}

		return nil
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
