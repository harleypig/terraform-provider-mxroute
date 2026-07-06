package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ForwarderResource satisfies the framework interfaces.
var (
	_ resource.Resource                = &ForwarderResource{}
	_ resource.ResourceWithConfigure   = &ForwarderResource{}
	_ resource.ResourceWithImportState = &ForwarderResource{}
)

// NewForwarderResource returns a new mxroute_forwarder resource.
func NewForwarderResource() resource.Resource {
	return &ForwarderResource{}
}

// ForwarderResource manages an email forwarder (alias) on a mail domain.
type ForwarderResource struct {
	client *Client
}

// ForwarderResourceModel maps the mxroute_forwarder schema to Go values.
type ForwarderResourceModel struct {
	Domain       types.String `tfsdk:"domain"`
	Alias        types.String `tfsdk:"alias"`
	Destinations types.List   `tfsdk:"destinations"`
	Email        types.String `tfsdk:"email"`
	ID           types.String `tfsdk:"id"`
}

// createForwarderRequest is the POST /domains/{domain}/forwarders body.
type createForwarderRequest struct {
	Alias        string   `json:"alias"`
	Destinations []string `json:"destinations"`
}

func (r *ForwarderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_forwarder"
}

func (r *ForwarderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an email forwarder (alias) on a mail domain. MXroute exposes no in-place update for a forwarder, so changing any attribute replaces the resource.",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain the forwarder belongs to (e.g. `example.com`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"alias": schema.StringAttribute{
				MarkdownDescription: "The local part of the forwarding address (e.g. `sales` for `sales@example.com`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"destinations": schema.ListAttribute{
				MarkdownDescription: "The email addresses mail to this alias is forwarded to. MXroute exposes no in-place update, so changing the destinations replaces the resource.",
				ElementType:         types.StringType,
				Required:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The full forwarding address (e.g. `sales@example.com`).",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier — `<domain>/<alias>`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ForwarderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResourceClient(req, resp); client != nil {
		r.client = client
	}
}

func (r *ForwarderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ForwarderResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var destinations []string

	resp.Diagnostics.Append(plan.Destinations.ElementsAs(ctx, &destinations, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := plan.Domain.ValueString()
	alias := plan.Alias.ValueString()

	body := createForwarderRequest{
		Alias:        alias,
		Destinations: destinations,
	}

	if err := r.client.Do(ctx, http.MethodPost, "/domains/"+domain+"/forwarders", body, nil); err != nil {
		resp.Diagnostics.AddError("Error creating forwarder", err.Error())

		return
	}

	// The create response carries no data; read the forwarder back to populate
	// the computed attributes.
	api, err := r.fetchForwarder(ctx, domain, alias)
	if err != nil {
		resp.Diagnostics.AddError("Error reading forwarder after create", err.Error())

		return
	}

	if api == nil {
		resp.Diagnostics.AddError("Error reading forwarder after create", fmt.Sprintf("forwarder %q on domain %q was not found immediately after creation", alias, domain))

		return
	}

	state, diags := forwarderModelFromAPI(ctx, domain, api)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ForwarderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ForwarderResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := state.Domain.ValueString()

	api, err := r.fetchForwarder(ctx, domain, state.Alias.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading forwarder", err.Error())

		return
	}

	if api == nil {
		resp.State.RemoveResource(ctx)

		return
	}

	newState, diags := forwarderModelFromAPI(ctx, domain, api)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Update refreshes the computed attributes. Every input attribute is
// RequiresReplace, so a plan never reaches Update with a changed value — it
// re-reads to keep state accurate.
func (r *ForwarderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ForwarderResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := plan.Domain.ValueString()
	alias := plan.Alias.ValueString()

	api, err := r.fetchForwarder(ctx, domain, alias)
	if err != nil {
		resp.Diagnostics.AddError("Error reading forwarder", err.Error())

		return
	}

	if api == nil {
		resp.Diagnostics.AddError("Error reading forwarder", fmt.Sprintf("forwarder %q on domain %q was not found", alias, domain))

		return
	}

	state, diags := forwarderModelFromAPI(ctx, domain, api)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ForwarderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ForwarderResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := state.Domain.ValueString()
	alias := state.Alias.ValueString()

	// A forwarder already gone is a successful delete.
	if err := r.client.Do(ctx, http.MethodDelete, "/domains/"+domain+"/forwarders/"+alias, nil, nil); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting forwarder", err.Error())

		return
	}
}

func (r *ForwarderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domain, alias, ok := strings.Cut(req.ID, "/")
	if !ok || domain == "" || alias == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier in the form `<domain>/<alias>`, got: %q.", req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), domain)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("alias"), alias)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// fetchForwarder lists a domain's forwarders and returns the one matching
// alias, or (nil, nil) when no such forwarder exists.
func (r *ForwarderResource) fetchForwarder(ctx context.Context, domain, alias string) (*Forwarder, error) {
	return fetchFromList(ctx, r.client, "/domains/"+domain+"/forwarders", func(f *Forwarder) bool { return f.Alias == alias })
}

// forwarderModelFromAPI maps an API forwarder onto the Terraform state model.
func forwarderModelFromAPI(ctx context.Context, domain string, api *Forwarder) (ForwarderResourceModel, diag.Diagnostics) {
	destinations, diags := types.ListValueFrom(ctx, types.StringType, api.Destinations)

	return ForwarderResourceModel{
		Domain:       types.StringValue(domain),
		Alias:        types.StringValue(api.Alias),
		Destinations: destinations,
		Email:        types.StringValue(api.Email),
		ID:           types.StringValue(domain + "/" + api.Alias),
	}, diags
}
