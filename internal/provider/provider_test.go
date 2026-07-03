package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"mxroute": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck skips an acceptance test when the live MXroute credentials
// are not present (e.g. the default CI gate), so a plain TF_ACC run without
// secrets never fails on a missing provider configuration.
func testAccPreCheck(t *testing.T) {
	t.Helper()

	for _, k := range []string{"MXROUTE_SERVER", "MXROUTE_USERNAME", "MXROUTE_API_KEY"} {
		if os.Getenv(k) == "" {
			t.Skipf("%s not set; skipping live-account acceptance test", k)
		}
	}
}
