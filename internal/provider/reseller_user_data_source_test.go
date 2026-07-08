package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccResellerUserDataSource requires a reseller account (in addition to
// TF_ACC + credentials) and a user named "someuser" on it; adjust the username
// to a user that exists when running against a real reseller account.
func TestAccResellerUserDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccResellerPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "mxroute_reseller_user" "test" {
  username = "someuser"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.mxroute_reseller_user.test", "username", "someuser"),
					resource.TestCheckResourceAttr("data.mxroute_reseller_user.test", "id", "someuser"),
					resource.TestCheckResourceAttrSet("data.mxroute_reseller_user.test", "quota_unlimited"),
				),
			},
		},
	})
}
