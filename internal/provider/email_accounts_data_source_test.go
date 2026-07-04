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
				// Create a throwaway domain, then list its (empty) mailboxes.
				Config: fmt.Sprintf(`
resource "mxroute_domain" "test" {
  domain = %[1]q
}

data "mxroute_email_accounts" "test" {
  domain = mxroute_domain.test.domain
}
`, domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.mxroute_email_accounts.test", "domain", domain),
					resource.TestCheckResourceAttr("data.mxroute_email_accounts.test", "id", domain),
					resource.TestCheckResourceAttrSet("data.mxroute_email_accounts.test", "accounts.#"),
				),
			},
		},
	})
}
