package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccCheckSpamBlacklistEntryDestroy confirms the blacklist entry is gone
// after the test. The parent domain is destroyed too, so a missing domain
// (NOT_FOUND) also counts as gone.
//
// NOTE: the GET /domains/{domain}/spam/blacklist response schema is
// unspecified in the OpenAPI; this assumes a bare array of strings (like the
// spam whitelist). When first running this test against the live account,
// verify the response shape and adjust the list type here and in the resource
// if it differs.
func testAccCheckSpamBlacklistEntryDestroy(t *testing.T, domain, entry string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := NewClient(ClientConfig{
			Server:   os.Getenv("MXROUTE_SERVER"),
			Username: os.Getenv("MXROUTE_USERNAME"),
			APIKey:   os.Getenv("MXROUTE_API_KEY"),
		})

		var list []string

		err := client.Do(t.Context(), "GET", "/domains/"+domain+"/spam/blacklist", nil, &list)
		if err != nil {
			if IsNotFound(err) {
				return nil
			}

			return fmt.Errorf("checking spam blacklist entry destroy: %w", err)
		}

		for _, e := range list {
			if e == entry {
				return fmt.Errorf("blacklist entry %q still exists on domain %q after destroy", entry, domain)
			}
		}

		return nil
	}
}

func TestAccSpamBlacklistEntryResource(t *testing.T) {
	domain := testAccTestDomain(t)
	entry := "spammer@example.net"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSpamBlacklistEntryDestroy(t, domain, entry),
		Steps: []resource.TestStep{
			{
				Config: testAccSpamBlacklistEntryResourceConfig(domain, entry),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_spam_blacklist_entry.test", "domain", domain),
					resource.TestCheckResourceAttr("mxroute_spam_blacklist_entry.test", "entry", entry),
					resource.TestCheckResourceAttr("mxroute_spam_blacklist_entry.test", "id", domain+"/"+entry),
				),
			},
			{
				ResourceName:      "mxroute_spam_blacklist_entry.test",
				ImportState:       true,
				ImportStateId:     domain + "/" + entry,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccSpamBlacklistEntryResourceConfig(domain, entry string) string {
	return fmt.Sprintf(`
resource "mxroute_domain" "test" {
  domain = %[1]q
}

resource "mxroute_spam_blacklist_entry" "test" {
  domain = mxroute_domain.test.domain
  entry  = %[2]q
}
`, domain, entry)
}
