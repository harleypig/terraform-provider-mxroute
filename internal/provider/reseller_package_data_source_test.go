package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccResellerPackageDataSource requires a reseller account (in addition to
// TF_ACC + credentials) and a package named "default" on it; adjust the name
// to a package that exists when running against a real reseller account.
func TestAccResellerPackageDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccResellerPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "mxroute_reseller_package" "test" {
  name = "default"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.mxroute_reseller_package.test", "name", "default"),
					resource.TestCheckResourceAttr("data.mxroute_reseller_package.test", "id", "default"),
					resource.TestCheckResourceAttrSet("data.mxroute_reseller_package.test", "settings.quota_unlimited"),
				),
			},
		},
	})
}
