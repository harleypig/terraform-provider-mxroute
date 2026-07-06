package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure SpamWhitelistEntryResource satisfies the framework interfaces.
var (
	_ resource.Resource                = &SpamWhitelistEntryResource{}
	_ resource.ResourceWithConfigure   = &SpamWhitelistEntryResource{}
	_ resource.ResourceWithImportState = &SpamWhitelistEntryResource{}
)

// NewSpamWhitelistEntryResource returns a new mxroute_spam_whitelist_entry
// resource.
func NewSpamWhitelistEntryResource() resource.Resource {
	return &SpamWhitelistEntryResource{}
}

// SpamWhitelistEntryResource manages a single entry in a domain's spam
// whitelist on the MXroute account.
type SpamWhitelistEntryResource struct {
	client *Client
}

// SpamWhitelistEntryResourceModel maps the mxroute_spam_whitelist_entry schema
// to Go values.
type SpamWhitelistEntryResourceModel struct {
	Domain types.String `tfsdk:"domain"`
	Entry  types.String `tfsdk:"entry"`
	ID     types.String `tfsdk:"id"`
}

// createSpamWhitelistEntryRequest is the
// POST /domains/{domain}/spam/whitelist body.
type createSpamWhitelistEntryRequest struct {
	Entry string `json:"entry"`
}

func (r *SpamWhitelistEntryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_spam_whitelist_entry"
}

func (r *SpamWhitelistEntryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a single entry in a domain's spam whitelist. The whitelist is a set of address patterns, so there is no in-place update: changing `domain` or `entry` replaces the resource.",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain whose spam whitelist the entry belongs to (e.g. `example.com`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"entry": schema.StringAttribute{
				MarkdownDescription: "The whitelist entry — an address or pattern to always accept (wildcards like `*@trusted.com` are supported).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier — `<domain>/<entry>`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *SpamWhitelistEntryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResourceClient(req, resp); client != nil {
		r.client = client
	}
}

func (r *SpamWhitelistEntryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SpamWhitelistEntryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := plan.Domain.ValueString()
	entry := plan.Entry.ValueString()

	body := createSpamWhitelistEntryRequest{Entry: entry}

	if err := r.client.Do(ctx, http.MethodPost, "/domains/"+domain+"/spam/whitelist", body, nil); err != nil {
		resp.Diagnostics.AddError("Error creating spam whitelist entry", err.Error())

		return
	}

	plan.ID = types.StringValue(domain + "/" + entry)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SpamWhitelistEntryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SpamWhitelistEntryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := state.Domain.ValueString()
	entry := state.Entry.ValueString()

	found, err := r.entryExists(ctx, domain, entry)
	if err != nil {
		resp.Diagnostics.AddError("Error reading spam whitelist entry", err.Error())

		return
	}

	if !found {
		resp.State.RemoveResource(ctx)

		return
	}

	state.ID = types.StringValue(domain + "/" + entry)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is unreachable: both attributes are RequiresReplace, so a changed
// value replaces the resource rather than updating it. It is defined only to
// satisfy the resource.Resource interface.
func (r *SpamWhitelistEntryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SpamWhitelistEntryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(plan.Domain.ValueString() + "/" + plan.Entry.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SpamWhitelistEntryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SpamWhitelistEntryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := state.Domain.ValueString()
	entry := state.Entry.ValueString()

	// An entry already gone is a successful delete.
	if err := r.client.Do(ctx, http.MethodDelete, "/domains/"+domain+"/spam/whitelist/"+entry, nil, nil); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting spam whitelist entry", err.Error())

		return
	}
}

func (r *SpamWhitelistEntryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domain, entry, found := strings.Cut(req.ID, "/")
	if !found || domain == "" || entry == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier in the form `<domain>/<entry>`, got: %q.", req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), domain)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("entry"), entry)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// entryExists GETs the domain's spam whitelist and reports whether entry is
// present. A missing domain (404) means the entry is gone too.
func (r *SpamWhitelistEntryResource) entryExists(ctx context.Context, domain, entry string) (bool, error) {
	var whitelist []string

	if err := r.client.Do(ctx, http.MethodGet, "/domains/"+domain+"/spam/whitelist", nil, &whitelist); err != nil {
		if IsNotFound(err) {
			return false, nil
		}

		return false, err
	}

	for _, candidate := range whitelist {
		if candidate == entry {
			return true, nil
		}
	}

	return false, nil
}
