package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccResellerPackageName returns the throwaway package name acceptance
// tests may create and destroy. Reseller packages exist only on a reseller
// account, which the harleydev test account is NOT, so the test skips unless
// MXROUTE_TEST_RESELLER_PACKAGE names a package to manage. Point it only at a
// disposable package on a real reseller account.
func testAccResellerPackageName(t *testing.T) string {
	t.Helper()

	name := os.Getenv("MXROUTE_TEST_RESELLER_PACKAGE")
	if name == "" {
		t.Skip("MXROUTE_TEST_RESELLER_PACKAGE not set; skipping reseller package acceptance test (requires a reseller account)")
	}

	return name
}

// testAccCheckResellerPackageDestroy confirms the package is gone after the
// test.
func testAccCheckResellerPackageDestroy(t *testing.T, name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := NewClient(ClientConfig{
			Server:   os.Getenv("MXROUTE_SERVER"),
			Username: os.Getenv("MXROUTE_USERNAME"),
			APIKey:   os.Getenv("MXROUTE_API_KEY"),
		})

		var api Package

		err := client.Do(t.Context(), "GET", "/reseller/packages/"+name, nil, &api)
		if err == nil {
			return fmt.Errorf("reseller package %q still exists after destroy", name)
		}

		if !IsNotFound(err) {
			return fmt.Errorf("checking reseller package destroy: %w", err)
		}

		return nil
	}
}

// TestAccResellerPackageResource exercises the full lifecycle of a reseller
// package. It REQUIRES A RESELLER ACCOUNT (set MXROUTE_TEST_RESELLER_PACKAGE),
// so it is skipped on the non-reseller harleydev test account. Being an
// account-level resource, it needs no parent domain.
func TestAccResellerPackageResource(t *testing.T) {
	name := testAccResellerPackageName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckResellerPackageDestroy(t, name),
		Steps: []resource.TestStep{
			{
				Config: testAccResellerPackageResourceConfig(name, "5", "10"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_reseller_package.test", "name", name),
					resource.TestCheckResourceAttr("mxroute_reseller_package.test", "id", name),
					resource.TestCheckResourceAttr("mxroute_reseller_package.test", "quota", "5"),
					resource.TestCheckResourceAttr("mxroute_reseller_package.test", "domains", "10"),
					resource.TestCheckResourceAttr("mxroute_reseller_package.test", "settings.quota_gb", "5"),
					resource.TestCheckResourceAttr("mxroute_reseller_package.test", "settings.domains", "10"),
				),
			},
			{
				Config: testAccResellerPackageResourceConfig(name, "unlimited", "20"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_reseller_package.test", "quota", "unlimited"),
					resource.TestCheckResourceAttr("mxroute_reseller_package.test", "domains", "20"),
					resource.TestCheckResourceAttr("mxroute_reseller_package.test", "settings.quota_unlimited", "true"),
					resource.TestCheckResourceAttr("mxroute_reseller_package.test", "settings.domains", "20"),
				),
			},
			{
				ResourceName:      "mxroute_reseller_package.test",
				ImportState:       true,
				ImportStateId:     name,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccResellerPackageResourceConfig(name, quota, domains string) string {
	return fmt.Sprintf(`
resource "mxroute_reseller_package" "test" {
  name    = %[1]q
  quota   = %[2]q
  domains = %[3]q
}
`, name, quota, domains)
}
