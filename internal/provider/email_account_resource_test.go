package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccEmailAccountUsername is the throwaway mailbox created and destroyed by
// the email account acceptance test.
const testAccEmailAccountUsername = "tfacctest"

// testAccCheckEmailAccountDestroy confirms the mailbox is gone after the test.
func testAccCheckEmailAccountDestroy(t *testing.T, domain, username string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		return checkGoneSingle[EmailAccount](t, "/domains/"+domain+"/email-accounts/"+username, fmt.Sprintf("email account %q on %q", username, domain))
	}
}

func TestAccEmailAccountResource(t *testing.T) {
	domain := testAccTestDomain(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEmailAccountDestroy(t, domain, testAccEmailAccountUsername),
		Steps: []resource.TestStep{
			{
				// The API enforces password complexity at create (uppercase,
				// lowercase, numbers, and a special character), so the fixture
				// must satisfy all four — a lowercase+digit-only value is
				// rejected with HTTP 400 VALIDATION_ERROR.
				Config: testAccEmailAccountResourceConfig(domain, testAccEmailAccountUsername, "Tf-Acc3ss-P4ss!", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_email_account.test", "domain", domain),
					resource.TestCheckResourceAttr("mxroute_email_account.test", "username", testAccEmailAccountUsername),
					resource.TestCheckResourceAttr("mxroute_email_account.test", "id", domain+"/"+testAccEmailAccountUsername),
					resource.TestCheckResourceAttr("mxroute_email_account.test", "email", testAccEmailAccountUsername+"@"+domain),
					// limit set at create must round-trip — before the create body
					// carried `limit`, this produced a provider-inconsistent result.
					resource.TestCheckResourceAttr("mxroute_email_account.test", "limit", "5000"),
					// The write-only password is never stored in state.
					resource.TestCheckNoResourceAttr("mxroute_email_account.test", "password_wo"),
				),
			},
			{
				// An existing mailbox updates with password_wo omitted (the
				// password is left unchanged): quota changes, the version
				// trigger is unchanged, so no password is required.
				Config: testAccEmailAccountResourceConfigNoPassword(domain, testAccEmailAccountUsername, 2048, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_email_account.test", "quota", "2048"),
					resource.TestCheckNoResourceAttr("mxroute_email_account.test", "password_wo"),
				),
			},
			{
				ResourceName:            "mxroute_email_account.test",
				ImportState:             true,
				ImportStateId:           domain + "/" + testAccEmailAccountUsername,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password_wo", "password_wo_version"},
			},
		},
	})
}

// TestAccEmailAccountResource_createRequiresPassword verifies that creating a
// mailbox without password_wo fails with a clear error — password_wo is
// optional in the schema (so an existing mailbox need not carry it) but the
// API requires a password to create one.
func TestAccEmailAccountResource_createRequiresPassword(t *testing.T) {
	domain := testAccTestDomain(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccEmailAccountResourceConfigCreateNoPassword(domain, testAccEmailAccountUsername),
				ExpectError: regexp.MustCompile("Missing password for new mailbox"),
			},
		},
	})
}

// TestAccEmailAccountResource_limitValidator asserts the plan-time upper bound
// on limit (spec maximum 9600). No PreCheck and no live account: the validator
// fires during plan, before any API call.
func TestAccEmailAccountResource_limitValidator(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "mxroute_email_account" "test" {
  domain      = "example.com"
  username    = "tfacctest"
  password_wo = "s3cret-p4ss"
  limit       = 99999
}`,
				ExpectError: regexp.MustCompile("at most 9600"),
			},
		},
	})
}

// TestAccEmailAccountResource_passwordLengthValidator asserts the plan-time
// minimum password length (spec minLength 8). Fires during plan.
func TestAccEmailAccountResource_passwordLengthValidator(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "mxroute_email_account" "test" {
  domain      = "example.com"
  username    = "tfacctest"
  password_wo = "short"
}`,
				ExpectError: regexp.MustCompile("at least 8"),
			},
		},
	})
}

// TestAccEmailAccountResource_passwordRotation exercises the write-only
// password rotate path: bumping password_wo_version with a new password_wo
// sends the new password on update (a write-only value cannot be diffed, so the
// version trigger is what drives it). The password never lands in state, so the
// assertions are that the rotation applies cleanly, the version advances, and
// password_wo stays absent from state.
func TestAccEmailAccountResource_passwordRotation(t *testing.T) {
	domain := testAccTestDomain(t)

	const username = "tfaccrotate"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckEmailAccountDestroy(t, domain, username),
		Steps: []resource.TestStep{
			{
				Config: testAccEmailAccountResourceConfig(domain, username, "Tf-Acc3ss-P4ss!", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_email_account.test", "username", username),
					resource.TestCheckResourceAttr("mxroute_email_account.test", "password_wo_version", "1"),
					resource.TestCheckNoResourceAttr("mxroute_email_account.test", "password_wo"),
				),
			},
			{
				// Rotate: new password, version bumped 1 -> 2.
				Config: testAccEmailAccountResourceConfig(domain, username, "N3w-R0tated-P4ss!", 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_email_account.test", "password_wo_version", "2"),
					resource.TestCheckNoResourceAttr("mxroute_email_account.test", "password_wo"),
				),
			},
		},
	})
}

func testAccEmailAccountResourceConfig(domain, username, password string, passwordVersion int) string {
	return fmt.Sprintf(`
resource "mxroute_domain" "test" {
  domain = %[1]q
}

resource "mxroute_email_account" "test" {
  domain              = mxroute_domain.test.domain
  username            = %[2]q
  password_wo         = %[3]q
  password_wo_version = %[4]d
  limit               = 5000
}
`, domain, username, password, passwordVersion)
}

// testAccEmailAccountResourceConfigNoPassword configures an existing mailbox
// with password_wo omitted (left unchanged) — the version trigger is held
// steady so no rotation is attempted.
func testAccEmailAccountResourceConfigNoPassword(domain, username string, quota, passwordVersion int) string {
	return fmt.Sprintf(`
resource "mxroute_domain" "test" {
  domain = %[1]q
}

resource "mxroute_email_account" "test" {
  domain              = mxroute_domain.test.domain
  username            = %[2]q
  quota               = %[3]d
  password_wo_version = %[4]d
}
`, domain, username, quota, passwordVersion)
}

// testAccEmailAccountResourceConfigCreateNoPassword configures a brand-new
// mailbox with no password_wo at all — used to assert create fails.
func testAccEmailAccountResourceConfigCreateNoPassword(domain, username string) string {
	return fmt.Sprintf(`
resource "mxroute_domain" "test" {
  domain = %[1]q
}

resource "mxroute_email_account" "test" {
  domain   = mxroute_domain.test.domain
  username = %[2]q
}
`, domain, username)
}
