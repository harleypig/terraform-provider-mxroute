package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccQuotaDataSource reads the account-wide quota. It is account-level, so
// it needs no parent domain and no CheckDestroy.
//
// XXX: ENVELOPE-UNVERIFIED — GET /quota may be UNENVELOPED (see the matching
// note in quota_data_source.go). This test is the live check: if the read
// succeeds but the attributes come back zero/empty (username unset,
// total_limit 0) against a real account that has data, /quota is almost
// certainly unenveloped and client.Do dropped the payload — the fix is a
// follow-up client change, not a change to this data source.
func TestAccQuotaDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccQuotaDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.mxroute_quota.test", "username"),
					resource.TestCheckResourceAttrSet("data.mxroute_quota.test", "id"),
					resource.TestCheckResourceAttrSet("data.mxroute_quota.test", "total_used"),
					resource.TestCheckResourceAttrSet("data.mxroute_quota.test", "total_limit"),
					resource.TestCheckResourceAttrSet("data.mxroute_quota.test", "breakdown.email"),
				),
			},
		},
	})
}

func testAccQuotaDataSourceConfig() string {
	return `
data "mxroute_quota" "test" {}
`
}
