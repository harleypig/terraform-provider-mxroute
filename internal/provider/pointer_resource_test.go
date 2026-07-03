package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccCheckPointerDestroy confirms the pointer is gone after the test. The
// parent domain is destroyed too, so a missing domain (NOT_FOUND) also counts
// as gone.
func testAccCheckPointerDestroy(t *testing.T, domain, pointer string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := NewClient(ClientConfig{
			Server:   os.Getenv("MXROUTE_SERVER"),
			Username: os.Getenv("MXROUTE_USERNAME"),
			APIKey:   os.Getenv("MXROUTE_API_KEY"),
		})

		var list []pointerAPIModel

		err := client.Do(t.Context(), "GET", "/domains/"+domain+"/pointers", nil, &list)
		if err != nil {
			if IsNotFound(err) {
				return nil
			}

			return fmt.Errorf("checking pointer destroy: %w", err)
		}

		for i := range list {
			if list[i].Pointer == pointer {
				return fmt.Errorf("pointer %q still exists on domain %q after destroy", pointer, domain)
			}
		}

		return nil
	}
}

func TestAccPointerResource(t *testing.T) {
	domain := testAccTestDomain(t)
	pointer := "www." + domain

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPointerDestroy(t, domain, pointer),
		Steps: []resource.TestStep{
			{
				Config: testAccPointerResourceConfig(domain, pointer),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("mxroute_pointer.test", "domain", domain),
					resource.TestCheckResourceAttr("mxroute_pointer.test", "pointer", pointer),
					resource.TestCheckResourceAttr("mxroute_pointer.test", "id", domain+"/"+pointer),
					resource.TestCheckResourceAttr("mxroute_pointer.test", "alias", "true"),
					resource.TestCheckResourceAttrSet("mxroute_pointer.test", "type"),
				),
			},
			{
				ResourceName:      "mxroute_pointer.test",
				ImportState:       true,
				ImportStateId:     domain + "/" + pointer,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccPointerResourceConfig(domain, pointer string) string {
	return fmt.Sprintf(`
resource "mxroute_domain" "test" {
  domain = %[1]q
}

resource "mxroute_pointer" "test" {
  domain  = mxroute_domain.test.domain
  pointer = %[2]q
}
`, domain, pointer)
}
