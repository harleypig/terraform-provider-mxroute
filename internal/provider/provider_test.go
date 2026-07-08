package provider

import (
	"fmt"
	"net/http"
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

// testAccResellerPreCheck skips a reseller acceptance test unless the account
// has reseller privileges. The reseller endpoints return HTTP 403 ("This
// endpoint requires reseller privileges") on a non-reseller account — as the
// harleydev test account is — so these tests opt in via MXROUTE_TEST_RESELLER.
// Set it only when running against a reseller-capable account.
func testAccResellerPreCheck(t *testing.T) {
	t.Helper()

	testAccPreCheck(t)

	if os.Getenv("MXROUTE_TEST_RESELLER") == "" {
		t.Skip("MXROUTE_TEST_RESELLER not set; skipping reseller acceptance test (requires a reseller account)")
	}
}

// newAccTestClient builds a Client from the live MXroute credentials in the
// environment — the shared constructor every acceptance-test CheckDestroy uses.
func newAccTestClient() *Client {
	return NewClient(ClientConfig{
		Server:   os.Getenv("MXROUTE_SERVER"),
		Username: os.Getenv("MXROUTE_USERNAME"),
		APIKey:   os.Getenv("MXROUTE_API_KEY"),
	})
}

// checkGoneSingle asserts that a single-object GET at path returns NOT_FOUND —
// i.e. the resource was destroyed. label names the resource in error messages.
func checkGoneSingle[T any](t *testing.T, path, label string) error {
	t.Helper()

	err := newAccTestClient().Do(t.Context(), http.MethodGet, path, nil, new(T))
	if err == nil {
		return fmt.Errorf("%s still exists after destroy", label)
	}

	if !IsNotFound(err) {
		return fmt.Errorf("checking %s destroy: %w", label, err)
	}

	return nil
}

// checkGoneInList asserts that no element of the []T GET at path matches — i.e.
// the resource was destroyed. A NOT_FOUND on the list itself (the parent gone)
// also counts as destroyed. label names the resource in error messages.
func checkGoneInList[T any](t *testing.T, path, label string, match func(*T) bool) error {
	t.Helper()

	var list []T

	err := newAccTestClient().Do(t.Context(), http.MethodGet, path, nil, &list)
	if err != nil {
		if IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("checking %s destroy: %w", label, err)
	}

	for i := range list {
		if match(&list[i]) {
			return fmt.Errorf("%s still exists after destroy", label)
		}
	}

	return nil
}
