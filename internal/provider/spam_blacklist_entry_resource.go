package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure SpamBlacklistEntryResource satisfies the framework interfaces.
var (
	_ resource.Resource                = &SpamBlacklistEntryResource{}
	_ resource.ResourceWithConfigure   = &SpamBlacklistEntryResource{}
	_ resource.ResourceWithImportState = &SpamBlacklistEntryResource{}
)

// NewSpamBlacklistEntryResource returns a new mxroute_spam_blacklist_entry
// resource.
func NewSpamBlacklistEntryResource() resource.Resource {
	return &SpamBlacklistEntryResource{}
}

// SpamBlacklistEntryResource manages a single entry in a domain's spam
// blacklist on the MXroute account.
type SpamBlacklistEntryResource struct {
	client *Client
}

// SpamBlacklistEntryResourceModel maps the mxroute_spam_blacklist_entry schema
// to Go values.
type SpamBlacklistEntryResourceModel struct {
	Domain types.String `tfsdk:"domain"`
	Entry  types.String `tfsdk:"entry"`
	ID     types.String `tfsdk:"id"`
}

// createSpamBlacklistEntryRequest is the POST
// /domains/{domain}/spam/blacklist body.
type createSpamBlacklistEntryRequest struct {
	Entry string `json:"entry"`
}

func (r *SpamBlacklistEntryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_spam_blacklist_entry"
}

func (r *SpamBlacklistEntryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a single entry in a mail domain's spam blacklist on the MXroute account. An entry is a sender address or domain whose mail is always rejected. MXroute exposes no in-place update for a blacklist entry, so changing any attribute replaces the resource.",
		Attributes: map[string]schema.Attribute{
			"domain": requiredReplaceString("The parent domain the blacklist entry belongs to (e.g. `example.com`)."),
			"entry":  requiredReplaceString("The blacklist entry — a sender address or domain to reject (e.g. `spammer@example.net`)."),
			"id":     computedIDAttribute("Resource identifier — `<domain>/<entry>`."),
		},
	}
}

func (r *SpamBlacklistEntryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResourceClient(req, resp); client != nil {
		r.client = client
	}
}

func (r *SpamBlacklistEntryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SpamBlacklistEntryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := plan.Domain.ValueString()
	entry := plan.Entry.ValueString()

	body := createSpamBlacklistEntryRequest{Entry: entry}

	if err := r.client.Do(ctx, http.MethodPost, "/domains/"+domain+"/spam/blacklist", body, nil); err != nil {
		resp.Diagnostics.AddError("Error creating spam blacklist entry", err.Error())

		return
	}

	// The entry carries no computed attributes, so state is derived entirely
	// from the plan; no read-back is needed.
	state := spamBlacklistEntryModel(domain, entry)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SpamBlacklistEntryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SpamBlacklistEntryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := state.Domain.ValueString()
	entry := state.Entry.ValueString()

	found, err := r.entryExists(ctx, domain, entry)
	if err != nil {
		resp.Diagnostics.AddError("Error reading spam blacklist entry", err.Error())

		return
	}

	if !found {
		resp.State.RemoveResource(ctx)

		return
	}

	newState := spamBlacklistEntryModel(domain, entry)

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Update is a no-op refresh. Both configurable attributes are RequiresReplace
// and the entry carries no computed attributes, so a plan never reaches Update
// with a changed value — it simply persists the plan to keep state accurate.
func (r *SpamBlacklistEntryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SpamBlacklistEntryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SpamBlacklistEntryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SpamBlacklistEntryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := state.Domain.ValueString()
	entry := state.Entry.ValueString()

	// An entry already gone is a successful delete.
	if err := r.client.Do(ctx, http.MethodDelete, "/domains/"+domain+"/spam/blacklist/"+entry, nil, nil); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting spam blacklist entry", err.Error())

		return
	}
}

func (r *SpamBlacklistEntryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domain, entry, found := strings.Cut(req.ID, "/")
	if !found || domain == "" || entry == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier of the form \"domain/entry\", got: %q", req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), domain)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("entry"), entry)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// entryExists lists the domain's spam blacklist and reports whether entry is
// present, returning (false, nil) when the domain or the blacklist does not
// exist.
//
// NOTE: the GET /domains/{domain}/spam/blacklist response schema is
// unspecified in the OpenAPI. This assumes it returns a bare array of strings
// like the spam whitelist. Verify against the live account (see the
// acceptance-test note); if the shape differs, adjust the list type here.
func (r *SpamBlacklistEntryResource) entryExists(ctx context.Context, domain, entry string) (bool, error) {
	found, err := fetchFromList(ctx, r.client, "/domains/"+domain+"/spam/blacklist", func(e *string) bool { return *e == entry })

	return found != nil, err
}

// spamBlacklistEntryModel builds the Terraform state model for one blacklist
// entry. The id is the `<domain>/<entry>` compound identifier.
func spamBlacklistEntryModel(domain, entry string) SpamBlacklistEntryResourceModel {
	return SpamBlacklistEntryResourceModel{
		Domain: types.StringValue(domain),
		Entry:  types.StringValue(entry),
		ID:     types.StringValue(domain + "/" + entry),
	}
}
