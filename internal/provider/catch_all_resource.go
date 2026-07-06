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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure CatchAllResource satisfies the framework interfaces.
var (
	_ resource.Resource                     = &CatchAllResource{}
	_ resource.ResourceWithConfigure        = &CatchAllResource{}
	_ resource.ResourceWithConfigValidators = &CatchAllResource{}
	_ resource.ResourceWithImportState      = &CatchAllResource{}
)

// catchAllTypes are the permitted values for the catch-all policy type.
var catchAllTypes = []string{"fail", "blackhole", "address"}

// NewCatchAllResource returns a new mxroute_catch_all resource.
func NewCatchAllResource() resource.Resource {
	return &CatchAllResource{}
}

// CatchAllResource manages a domain's catch-all policy at MXroute. The policy
// is a per-domain singleton with no create or delete verb: it always exists,
// so the resource models its lifecycle as a PATCH of the desired policy and a
// reset to the "fail" default on delete.
type CatchAllResource struct {
	client *Client
}

// CatchAllResourceModel maps the mxroute_catch_all schema to Go values.
type CatchAllResourceModel struct {
	Domain      types.String `tfsdk:"domain"`
	Type        types.String `tfsdk:"type"`
	Address     types.String `tfsdk:"address"`
	Description types.String `tfsdk:"description"`
	ID          types.String `tfsdk:"id"`
}

// patchCatchAllRequest is the PATCH /domains/{domain}/catch-all body. Address
// is sent only when the policy type is "address".
type patchCatchAllRequest struct {
	Type    string  `json:"type"`
	Address *string `json:"address,omitempty"`
}

func (r *CatchAllResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_catch_all"
}

func (r *CatchAllResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the catch-all policy for a domain hosted at MXroute. The policy is a per-domain singleton, so this resource has no create or delete API call: it PATCHes the desired policy and resets to the `fail` default when destroyed. Changing `domain` replaces the resource.",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain whose catch-all policy this manages (e.g. `example.com`). Changing this replaces the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The catch-all policy: `fail` (reject mail to unknown addresses), `blackhole` (silently discard it), or `address` (deliver it to `address`).",
				Required:            true,
				Validators: []validator.String{
					catchAllTypeValidator{},
				},
			},
			"address": schema.StringAttribute{
				MarkdownDescription: "The mailbox catch-all mail is delivered to. Required when `type` is `address`, and must be omitted otherwise.",
				Optional:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Human-readable description of the catch-all policy, as reported by the API.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier — the domain name.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// ConfigValidators enforces the cross-field rule that `address` is set exactly
// when `type` is "address".
func (r *CatchAllResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		catchAllAddressValidator{},
	}
}

func (r *CatchAllResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResourceClient(req, resp); client != nil {
		r.client = client
	}
}

// Create PATCHes the desired catch-all policy — the singleton has no create
// verb — then reads it back to populate the computed attributes.
func (r *CatchAllResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CatchAllResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := plan.Domain.ValueString()

	if err := r.client.Do(ctx, http.MethodPatch, catchAllPath(domain), catchAllRequestFromPlan(plan), nil); err != nil {
		resp.Diagnostics.AddError("Error setting catch-all policy", err.Error())

		return
	}

	api, err := r.fetchCatchAll(ctx, domain)
	if err != nil {
		resp.Diagnostics.AddError("Error reading catch-all policy after create", err.Error())

		return
	}

	if api == nil {
		resp.Diagnostics.AddError("Error reading catch-all policy after create", fmt.Sprintf("catch-all policy for %q was not found immediately after being set", domain))

		return
	}

	state := catchAllStateFromAPI(api, domain)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CatchAllResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CatchAllResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := state.Domain.ValueString()

	api, err := r.fetchCatchAll(ctx, domain)
	if err != nil {
		resp.Diagnostics.AddError("Error reading catch-all policy", err.Error())

		return
	}

	if api == nil {
		resp.State.RemoveResource(ctx)

		return
	}

	newState := catchAllStateFromAPI(api, domain)

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Update PATCHes the new policy and reads it back to refresh state.
func (r *CatchAllResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CatchAllResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := plan.Domain.ValueString()

	if err := r.client.Do(ctx, http.MethodPatch, catchAllPath(domain), catchAllRequestFromPlan(plan), nil); err != nil {
		resp.Diagnostics.AddError("Error updating catch-all policy", err.Error())

		return
	}

	api, err := r.fetchCatchAll(ctx, domain)
	if err != nil {
		resp.Diagnostics.AddError("Error reading catch-all policy after update", err.Error())

		return
	}

	if api == nil {
		resp.Diagnostics.AddError("Error reading catch-all policy after update", fmt.Sprintf("catch-all policy for %q was not found after update", domain))

		return
	}

	state := catchAllStateFromAPI(api, domain)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete resets the catch-all policy to the "fail" default — the singleton has
// no delete verb, so removing the resource restores the default behaviour.
func (r *CatchAllResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CatchAllResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The domain being gone already leaves no catch-all policy to reset.
	if err := r.client.Do(ctx, http.MethodPatch, catchAllPath(state.Domain.ValueString()), patchCatchAllRequest{Type: "fail"}, nil); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Error resetting catch-all policy", err.Error())

		return
	}
}

func (r *CatchAllResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// fetchCatchAll GETs a domain's catch-all policy, returning (nil, nil) when
// the domain (and thus its policy) does not exist.
func (r *CatchAllResource) fetchCatchAll(ctx context.Context, domain string) (*CatchAll, error) {
	return fetchOne[CatchAll](ctx, r.client, catchAllPath(domain))
}

// catchAllPath is the API path for a domain's catch-all policy.
func catchAllPath(domain string) string {
	return "/domains/" + domain + "/catch-all"
}

// catchAllRequestFromPlan builds the PATCH body from the plan, sending the
// address only for the "address" policy.
func catchAllRequestFromPlan(plan CatchAllResourceModel) patchCatchAllRequest {
	body := patchCatchAllRequest{Type: plan.Type.ValueString()}

	if plan.Type.ValueString() == "address" && !plan.Address.IsNull() {
		value := plan.Address.ValueString()
		body.Address = &value
	}

	return body
}

// catchAllStateFromAPI maps an API catch-all policy onto the Terraform state
// model. Address is null unless the API reports one (only the "address" type).
func catchAllStateFromAPI(api *CatchAll, domain string) CatchAllResourceModel {
	address := types.StringNull()
	if api.Address != nil {
		address = types.StringValue(*api.Address)
	}

	return CatchAllResourceModel{
		Domain:      types.StringValue(domain),
		Type:        types.StringValue(api.Type),
		Address:     address,
		Description: types.StringValue(api.Description),
		ID:          types.StringValue(domain),
	}
}

// catchAllTypeValidator validates that the catch-all type is one of the
// permitted policy values.
type catchAllTypeValidator struct{}

func (v catchAllTypeValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("value must be one of: %s", strings.Join(catchAllTypes, ", "))
}

func (v catchAllTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v catchAllTypeValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	for _, allowed := range catchAllTypes {
		if value == allowed {
			return
		}
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid catch-all type",
		fmt.Sprintf("Expected one of %s, got: %q.", strings.Join(catchAllTypes, ", "), value),
	)
}

// catchAllAddressValidator enforces that `address` is set exactly when `type`
// is "address" — required for that type, and forbidden for any other.
type catchAllAddressValidator struct{}

func (v catchAllAddressValidator) Description(ctx context.Context) string {
	return "address must be set when type is \"address\" and omitted otherwise"
}

func (v catchAllAddressValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v catchAllAddressValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data CatchAllResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A rule that depends on an unknown value cannot be checked yet.
	if data.Type.IsUnknown() || data.Address.IsUnknown() {
		return
	}

	isAddressType := data.Type.ValueString() == "address"
	hasAddress := !data.Address.IsNull()

	if isAddressType && !hasAddress {
		resp.Diagnostics.AddAttributeError(
			path.Root("address"),
			"Missing address",
			"`address` is required when `type` is \"address\".",
		)
	}

	if !isAddressType && hasAddress {
		resp.Diagnostics.AddAttributeError(
			path.Root("address"),
			"Unexpected address",
			"`address` may only be set when `type` is \"address\".",
		)
	}
}
