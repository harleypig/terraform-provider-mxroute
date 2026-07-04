package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccResellerPackagesDataSource requires a reseller account (in addition
// to TF_ACC + credentials).
func TestAccResellerPackagesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "mxroute_reseller_packages" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.mxroute_reseller_packages.all", "names.#"),
					resource.TestCheckResourceAttr("data.mxroute_reseller_packages.all", "id", "reseller_packages"),
				),
			},
		},
	})
}
