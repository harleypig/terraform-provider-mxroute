package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccEmailQuotaDataSource reads per-mailbox usage. It is account-level, so
// it needs no parent domain and no CheckDestroy.
//
// XXX: ENVELOPE-UNVERIFIED — GET /quota/email may be UNENVELOPED (see the
// matching note in email_quota_data_source.go). This test is the live check:
// if the read succeeds but the attributes come back zero/empty (username
// unset, accounts empty) against a real account that has mailboxes,
// /quota/email is almost certainly unenveloped and client.Do dropped the
// payload — the fix is a follow-up client change, not a change to this data
// source.
func TestAccEmailQuotaDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEmailQuotaDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.mxroute_email_quota.test", "username"),
					resource.TestCheckResourceAttrSet("data.mxroute_email_quota.test", "id"),
					resource.TestCheckResourceAttrSet("data.mxroute_email_quota.test", "accounts.#"),
				),
			},
		},
	})
}

func testAccEmailQuotaDataSourceConfig() string {
	return `
data "mxroute_email_quota" "test" {}
`
}
