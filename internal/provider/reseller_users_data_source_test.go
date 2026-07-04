package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccResellerUsersDataSource requires a reseller account (in addition to
// TF_ACC + credentials).
func TestAccResellerUsersDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "mxroute_reseller_users" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.mxroute_reseller_users.all", "usernames.#"),
					resource.TestCheckResourceAttr("data.mxroute_reseller_users.all", "id", "reseller_users"),
				),
			},
		},
	})
}
