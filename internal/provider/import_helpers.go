package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// importSingleKey imports a resource whose import ID is a single opaque key,
// setting both the named key attribute and "id" to req.ID.
func importSingleKey(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse, keyAttr string) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(keyAttr), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// importTwoPart imports a resource whose import ID is "<first>/<second>". It
// splits on the first slash, requires both halves non-empty, and sets
// firstAttr, secondAttr, and "id" — giving every compound-key resource one
// consistent identifier format and error message.
func importTwoPart(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse, firstAttr, secondAttr string) {
	first, second, found := strings.Cut(req.ID, "/")
	if !found || first == "" || second == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier in the form `<%s>/<%s>`, got: %q.", firstAttr, secondAttr, req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(firstAttr), first)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(secondAttr), second)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
