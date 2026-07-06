package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccResellerUserResource_createRequiresPassword verifies that creating a
// reseller user without password_wo fails with a clear error — password_wo is
// now Optional (so an existing user need not carry it) but the API requires a
// password to create one. The create-time guard fires before any API call, so
// this needs no live reseller account.
func TestAccResellerUserResource_createRequiresPassword(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "mxroute_reseller_user" "test" {
  username = "tfacct"
  email    = "tfacct@example.com"
  package  = "default"
}`,
				ExpectError: regexp.MustCompile("Missing password for new reseller user"),
			},
		},
	})
}

// TestAccResellerUserResource_passwordLengthValidator asserts the plan-time
// minimum password length (spec minLength 8).
func TestAccResellerUserResource_passwordLengthValidator(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "mxroute_reseller_user" "test" {
  username    = "tfacct"
  email       = "tfacct@example.com"
  package     = "default"
  password_wo = "short"
}`,
				ExpectError: regexp.MustCompile("at least 8"),
			},
		},
	})
}

// testAccResellerUser returns the throwaway reseller username acceptance tests
// may create and destroy. Reseller management is not available on the test
// account, so this test is skipped unconditionally; the config is kept correct
// against the spec for the day a reseller-capable account is wired up.
func testAccResellerUser(t *testing.T) string {
	t.Helper()

	t.Skip("mxroute_reseller_user requires reseller API access, unavailable on the test account")

	username := os.Getenv("MXROUTE_TEST_RESELLER_USERNAME")
	if username == "" {
		t.Skip("MXROUTE_TEST_RESELLER_USERNAME not set; skipping reseller user acceptance test")
	}

	return username
}

// testAccCheckResellerUserDestroy confirms the reseller user is gone after the
// test.
func testAccCheckResellerUserDestroy(t *testing.T, username string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		return checkGoneSingle[ResellerUser](t, "/reseller/users/"+username, fmt.Sprintf("reseller user %q", username))
	}
}

func TestAccResellerUserResource(t *testing.T) {
	username := testAccResellerUser(t)

	pkg := os.Getenv("MXROUTE_TEST_RESELLER_PACKAGE")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckResellerUserDestroy(t, username),
		Steps: []resource.TestStep{
			{
				Config: testAccResellerUserResourceConfig(username, pkg, "S3cret-pass", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_reseller_user.test", "username", username),
					resource.TestCheckResourceAttr("mxroute_reseller_user.test", "id", username),
					resource.TestCheckResourceAttr("mxroute_reseller_user.test", "package", pkg),
					resource.TestCheckResourceAttrSet("mxroute_reseller_user.test", "suspended"),
				),
			},
			{
				Config: testAccResellerUserResourceConfigSuspended(username, pkg, "S3cret-pass", 1, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_reseller_user.test", "suspended", "true"),
				),
			},
			{
				ResourceName:            "mxroute_reseller_user.test",
				ImportState:             true,
				ImportStateId:           username,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password_wo", "password_wo_version"},
			},
		},
	})
}

func testAccResellerUserResourceConfig(username, pkg, password string, version int) string {
	return fmt.Sprintf(`
resource "mxroute_reseller_user" "test" {
  username            = %[1]q
  email               = "%[1]s@example.com"
  package             = %[2]q
  password_wo         = %[3]q
  password_wo_version = %[4]d
}
`, username, pkg, password, version)
}

func testAccResellerUserResourceConfigSuspended(username, pkg, password string, version int, suspended bool) string {
	return fmt.Sprintf(`
resource "mxroute_reseller_user" "test" {
  username            = %[1]q
  email               = "%[1]s@example.com"
  package             = %[2]q
  password_wo         = %[3]q
  password_wo_version = %[4]d
  suspended           = %[5]t
}
`, username, pkg, password, version, suspended)
}
