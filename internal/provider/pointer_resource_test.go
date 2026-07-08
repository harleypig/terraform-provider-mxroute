package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccCheckPointerDestroy confirms the pointer is gone after the test. The
// parent domain is destroyed too, so a missing domain (NOT_FOUND) also counts
// as gone.
func testAccCheckPointerDestroy(t *testing.T, domain, pointer string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		return checkGoneInList(t, "/domains/"+domain+"/pointers", fmt.Sprintf("pointer %q on domain %q", pointer, domain), func(p *DomainPointer) bool { return p.Pointer == pointer })
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
					// Confirms the Domain.pointers decode against a live, populated
					// response: with exactly one pointer on the fresh domain, the
					// domain's pointers list must hold its name. Before the decode
					// fix this GET failed outright ("cannot unmarshal object into
					// []string"). pointers.0 asserts the object's keys are the
					// pointer names (the assumed shape) — if that assertion fails
					// live, the names live in the values instead; adjust
					// decodePointerNames accordingly.
					resource.TestCheckResourceAttr("data.mxroute_domain.test", "pointers.#", "1"),
					resource.TestCheckResourceAttr("data.mxroute_domain.test", "pointers.0", pointer),
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

# Read the domain back after the pointer exists so its computed pointers
# list is populated — this is what exercises the Domain.pointers decode.
data "mxroute_domain" "test" {
  domain     = mxroute_domain.test.domain
  depends_on = [mxroute_pointer.test]
}
`, domain, pointer)
}
