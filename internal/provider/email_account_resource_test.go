package provider

import (
	"fmt"
	"os"
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
		client := NewClient(ClientConfig{
			Server:   os.Getenv("MXROUTE_SERVER"),
			Username: os.Getenv("MXROUTE_USERNAME"),
			APIKey:   os.Getenv("MXROUTE_API_KEY"),
		})

		var api emailAccountAPIModel

		err := client.Do(t.Context(), "GET", "/domains/"+domain+"/email-accounts/"+username, nil, &api)
		if err == nil {
			return fmt.Errorf("email account %q on %q still exists after destroy", username, domain)
		}

		if !IsNotFound(err) {
			return fmt.Errorf("checking email account destroy: %w", err)
		}

		return nil
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
				Config: testAccEmailAccountResourceConfig(domain, testAccEmailAccountUsername, "s3cret-p4ss", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_email_account.test", "domain", domain),
					resource.TestCheckResourceAttr("mxroute_email_account.test", "username", testAccEmailAccountUsername),
					resource.TestCheckResourceAttr("mxroute_email_account.test", "id", domain+"/"+testAccEmailAccountUsername),
					resource.TestCheckResourceAttr("mxroute_email_account.test", "email", testAccEmailAccountUsername+"@"+domain),
					resource.TestCheckResourceAttrSet("mxroute_email_account.test", "limit"),
					// The write-only password is never stored in state.
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
}
`, domain, username, password, passwordVersion)
}
