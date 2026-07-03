package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccVerificationKeyDataSource reads the account ownership-verification
// record. It is account-level, so it needs no parent domain and no
// CheckDestroy.
//
// XXX: ENVELOPE-UNVERIFIED — GET /verification-key may be UNENVELOPED (see the
// matching note in verification_key_data_source.go). This test is the live
// check: if the read succeeds but the attributes come back zero/empty (key
// unset, record.type unset) against a real account, /verification-key is
// almost certainly unenveloped and client.Do dropped the payload — the fix is
// a follow-up client change, not a change to this data source.
func TestAccVerificationKeyDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVerificationKeyDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.mxroute_verification_key.test", "key"),
					resource.TestCheckResourceAttrSet("data.mxroute_verification_key.test", "id"),
					resource.TestCheckResourceAttrSet("data.mxroute_verification_key.test", "record.type"),
					resource.TestCheckResourceAttrSet("data.mxroute_verification_key.test", "record.name"),
					resource.TestCheckResourceAttrSet("data.mxroute_verification_key.test", "record.value"),
				),
			},
		},
	})
}

func testAccVerificationKeyDataSourceConfig() string {
	return `
data "mxroute_verification_key" "test" {}
`
}
