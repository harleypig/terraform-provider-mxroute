package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEmailAccountsDataSource(t *testing.T) {
	domain := testAccTestDomain(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDomainDestroy(t, domain),
		Steps: []resource.TestStep{
			{
				// Create a throwaway domain with one mailbox, then list the
				// domain's mailboxes and assert the element content — not just
				// the count. With exactly one mailbox, accounts.0 is
				// deterministic; quota is asserted because the resource test
				// already proves it round-trips through the API.
				Config: fmt.Sprintf(`
resource "mxroute_domain" "test" {
  domain = %[1]q
}

resource "mxroute_email_account" "test" {
  domain              = mxroute_domain.test.domain
  username            = %[2]q
  password_wo         = "Tf-Acc3ss-P4ss!"
  password_wo_version = 1
  quota               = 1024
}

data "mxroute_email_accounts" "test" {
  domain     = mxroute_domain.test.domain
  depends_on = [mxroute_email_account.test]
}
`, domain, testAccEmailAccountUsername),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.mxroute_email_accounts.test", "domain", domain),
					resource.TestCheckResourceAttr("data.mxroute_email_accounts.test", "id", domain),
					resource.TestCheckResourceAttr("data.mxroute_email_accounts.test", "accounts.#", "1"),
					resource.TestCheckResourceAttr("data.mxroute_email_accounts.test", "accounts.0.username", testAccEmailAccountUsername),
					resource.TestCheckResourceAttr("data.mxroute_email_accounts.test", "accounts.0.email", testAccEmailAccountUsername+"@"+domain),
					resource.TestCheckResourceAttr("data.mxroute_email_accounts.test", "accounts.0.quota", "1024"),
				),
			},
		},
	})
}
