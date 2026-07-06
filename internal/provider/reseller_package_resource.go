package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ResellerPackageResource satisfies the framework interfaces.
var (
	_ resource.Resource                = &ResellerPackageResource{}
	_ resource.ResourceWithConfigure   = &ResellerPackageResource{}
	_ resource.ResourceWithImportState = &ResellerPackageResource{}
)

// NewResellerPackageResource returns a new mxroute_reseller_package resource.
func NewResellerPackageResource() resource.Resource {
	return &ResellerPackageResource{}
}

// ResellerPackageResource manages a reseller package on the MXroute account.
// It is only usable on a reseller account.
type ResellerPackageResource struct {
	client *Client
}

// ResellerPackageResourceModel maps the mxroute_reseller_package schema to Go
// values. The limit attributes are strings because the API accepts strings
// ("5", "unlimited") and reflects them as typed settings; the configured
// string is the source of truth, while the computed settings object surfaces
// the typed values MXroute parsed from them.
type ResellerPackageResourceModel struct {
	Name            types.String `tfsdk:"name"`
	Quota           types.String `tfsdk:"quota"`
	Domains         types.String `tfsdk:"domains"`
	EmailAccounts   types.String `tfsdk:"email_accounts"`
	EmailForwarders types.String `tfsdk:"email_forwarders"`
	DomainPointers  types.String `tfsdk:"domain_pointers"`
	Settings        types.Object `tfsdk:"settings"`
	ID              types.String `tfsdk:"id"`
}

// resellerPackageSettingsModel maps the computed settings object to Go values.
type resellerPackageSettingsModel struct {
	QuotaGB         types.Float64 `tfsdk:"quota_gb"`
	QuotaUnlimited  types.Bool    `tfsdk:"quota_unlimited"`
	Domains         types.Int64   `tfsdk:"domains"`
	EmailAccounts   types.Int64   `tfsdk:"email_accounts"`
	EmailForwarders types.Int64   `tfsdk:"email_forwarders"`
	DomainPointers  types.Int64   `tfsdk:"domain_pointers"`
}

// resellerPackageSettingsAttrTypes is the attribute-type map for the computed
// settings object.
var resellerPackageSettingsAttrTypes = map[string]attr.Type{
	"quota_gb":         types.Float64Type,
	"quota_unlimited":  types.BoolType,
	"domains":          types.Int64Type,
	"email_accounts":   types.Int64Type,
	"email_forwarders": types.Int64Type,
	"domain_pointers":  types.Int64Type,
}

// createResellerPackageRequest is the POST /reseller/packages body. Every
// limit field is an optional string, so nil values are omitted.
type createResellerPackageRequest struct {
	Name            string  `json:"name"`
	Quota           *string `json:"quota,omitempty"`
	Domains         *string `json:"domains,omitempty"`
	EmailAccounts   *string `json:"email_accounts,omitempty"`
	EmailForwarders *string `json:"email_forwarders,omitempty"`
	DomainPointers  *string `json:"domain_pointers,omitempty"`
}

// updateResellerPackageRequest is the PATCH /reseller/packages/{name} body.
// The name is carried in the path, so it is not part of the payload; every
// limit field is optional.
type updateResellerPackageRequest struct {
	Quota           *string `json:"quota,omitempty"`
	Domains         *string `json:"domains,omitempty"`
	EmailAccounts   *string `json:"email_accounts,omitempty"`
	EmailForwarders *string `json:"email_forwarders,omitempty"`
	DomainPointers  *string `json:"domain_pointers,omitempty"`
}

func (r *ResellerPackageResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reseller_package"
}

func (r *ResellerPackageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a reseller package on the MXroute account. Requires a reseller account. Limit attributes are strings (for example `\"5\"` or `\"unlimited\"`); the configured value is the source of truth, and the computed `settings` object exposes the typed limits MXroute parsed from them. Each limit's create-time default (stated per attribute below) comes from the [MXroute API](https://api.mxroute.com/docs).",
		Attributes: map[string]schema.Attribute{
			"name": requiredReplaceString("The package name. MXroute keys a package by name and exposes no rename, so changing `name` replaces the resource."),
			"quota": schema.StringAttribute{
				MarkdownDescription: "Storage quota granted by the package, as a string (for example `\"5\"` for 5 GB, or `\"unlimited\"`). When unset, it is populated from the package's current settings; the API default for a new package is `\"1\"` (1 GB).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domains": schema.StringAttribute{
				MarkdownDescription: "Maximum number of domains, as a string (for example `\"10\"` or `\"unlimited\"`). When unset, it is populated from the package's current settings; the API default for a new package is `\"1\"`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"email_accounts": schema.StringAttribute{
				MarkdownDescription: "Maximum number of email accounts, as a string (for example `\"50\"` or `\"unlimited\"`). When unset, it is populated from the package's current settings; the API default for a new package is `\"100\"`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"email_forwarders": schema.StringAttribute{
				MarkdownDescription: "Maximum number of email forwarders, as a string (for example `\"50\"` or `\"unlimited\"`). When unset, it is populated from the package's current settings; the API default for a new package is `\"100\"`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_pointers": schema.StringAttribute{
				MarkdownDescription: "Maximum number of domain pointers, as a string (for example `\"10\"` or `\"unlimited\"`). When unset, it is populated from the package's current settings; the API default for a new package is `\"10\"`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"settings": schema.SingleNestedAttribute{
				MarkdownDescription: "The typed limits MXroute parsed from the configured string values. A null sub-field means the limit is unlimited. Recomputed from the API after each apply, so it is known-after-apply when the limits change.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"quota_gb": schema.Float64Attribute{
						MarkdownDescription: "Storage quota in gigabytes; null when the quota is unlimited.",
						Computed:            true,
					},
					"quota_unlimited": schema.BoolAttribute{
						MarkdownDescription: "Whether the storage quota is unlimited.",
						Computed:            true,
					},
					"domains": schema.Int64Attribute{
						MarkdownDescription: "Maximum number of domains; null when unlimited.",
						Computed:            true,
					},
					"email_accounts": schema.Int64Attribute{
						MarkdownDescription: "Maximum number of email accounts; null when unlimited.",
						Computed:            true,
					},
					"email_forwarders": schema.Int64Attribute{
						MarkdownDescription: "Maximum number of email forwarders; null when unlimited.",
						Computed:            true,
					},
					"domain_pointers": schema.Int64Attribute{
						MarkdownDescription: "Maximum number of domain pointers; null when unlimited.",
						Computed:            true,
					},
				},
			},
			"id": computedIDAttribute("Resource identifier — the package name."),
		},
	}
}

func (r *ResellerPackageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResourceClient(req, resp); client != nil {
		r.client = client
	}
}

func (r *ResellerPackageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResellerPackageResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()

	body := createResellerPackageRequest{
		Name:            name,
		Quota:           rpStringPtr(plan.Quota),
		Domains:         rpStringPtr(plan.Domains),
		EmailAccounts:   rpStringPtr(plan.EmailAccounts),
		EmailForwarders: rpStringPtr(plan.EmailForwarders),
		DomainPointers:  rpStringPtr(plan.DomainPointers),
	}

	if err := r.client.Do(ctx, http.MethodPost, "/reseller/packages", body, nil); err != nil {
		resp.Diagnostics.AddError("Error creating reseller package", err.Error())

		return
	}

	// The create response is partial; read the package back to populate the
	// computed attributes.
	api, err := r.fetchPackage(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Error reading reseller package after create", err.Error())

		return
	}

	if api == nil {
		resp.Diagnostics.AddError("Error reading reseller package after create", fmt.Sprintf("package %q was not found immediately after creation", name))

		return
	}

	state, diags := resellerPackageModelFromAPI(ctx, api, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ResellerPackageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResellerPackageResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	api, err := r.fetchPackage(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading reseller package", err.Error())

		return
	}

	if api == nil {
		resp.State.RemoveResource(ctx)

		return
	}

	newState, diags := resellerPackageModelFromAPI(ctx, api, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Update applies the changed limit fields, then re-reads the package to keep
// the computed attributes accurate. The name attribute is RequiresReplace, so
// a plan never reaches Update with a changed name.
func (r *ResellerPackageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResellerPackageResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()

	body := updateResellerPackageRequest{
		Quota:           rpStringPtr(plan.Quota),
		Domains:         rpStringPtr(plan.Domains),
		EmailAccounts:   rpStringPtr(plan.EmailAccounts),
		EmailForwarders: rpStringPtr(plan.EmailForwarders),
		DomainPointers:  rpStringPtr(plan.DomainPointers),
	}

	if err := r.client.Do(ctx, http.MethodPatch, "/reseller/packages/"+name, body, nil); err != nil {
		resp.Diagnostics.AddError("Error updating reseller package", err.Error())

		return
	}

	api, err := r.fetchPackage(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Error reading reseller package", err.Error())

		return
	}

	if api == nil {
		resp.Diagnostics.AddError("Error reading reseller package", fmt.Sprintf("package %q was not found", name))

		return
	}

	newState, diags := resellerPackageModelFromAPI(ctx, api, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *ResellerPackageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResellerPackageResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A package already gone is a successful delete.
	if err := r.client.Do(ctx, http.MethodDelete, "/reseller/packages/"+state.Name.ValueString(), nil, nil); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting reseller package", err.Error())

		return
	}
}

func (r *ResellerPackageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// fetchPackage GETs a reseller package, returning (nil, nil) when it does not
// exist.
func (r *ResellerPackageResource) fetchPackage(ctx context.Context, name string) (*Package, error) {
	return fetchOne[Package](ctx, r.client, "/reseller/packages/"+name)
}

// resellerPackageModelFromAPI maps an API package onto the Terraform state
// model. The limit strings are the source of truth: a value the caller
// supplied (in input, a plan or prior state) is preserved as-is, and only an
// unset limit is derived from the typed settings the GET returned. The typed
// settings are always refreshed into the computed settings object.
func resellerPackageModelFromAPI(ctx context.Context, api *Package, input ResellerPackageResourceModel) (ResellerPackageResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	settingsModel := resellerPackageSettingsModel{
		QuotaGB:         rpFloat64Value(api.Settings.QuotaGB),
		QuotaUnlimited:  types.BoolValue(api.Settings.QuotaUnlimited),
		Domains:         rpInt64Value(api.Settings.Domains),
		EmailAccounts:   rpInt64Value(api.Settings.EmailAccounts),
		EmailForwarders: rpInt64Value(api.Settings.EmailForwarders),
		DomainPointers:  rpInt64Value(api.Settings.DomainPointers),
	}

	settingsObj, d := types.ObjectValueFrom(ctx, resellerPackageSettingsAttrTypes, settingsModel)
	diags.Append(d...)

	model := ResellerPackageResourceModel{
		Name:            types.StringValue(api.Name),
		Quota:           rpResolveString(input.Quota, rpQuotaString(api.Settings)),
		Domains:         rpResolveString(input.Domains, rpCountString(api.Settings.Domains)),
		EmailAccounts:   rpResolveString(input.EmailAccounts, rpCountString(api.Settings.EmailAccounts)),
		EmailForwarders: rpResolveString(input.EmailForwarders, rpCountString(api.Settings.EmailForwarders)),
		DomainPointers:  rpResolveString(input.DomainPointers, rpCountString(api.Settings.DomainPointers)),
		Settings:        settingsObj,
		ID:              types.StringValue(api.Name),
	}

	return model, diags
}

// rpStringPtr returns a pointer to the string value, or nil when the value is
// null or unknown, so unset limits are omitted from a request body.
func rpStringPtr(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}

	s := v.ValueString()

	return &s
}

// rpResolveString keeps a caller-supplied limit string as the source of truth,
// falling back to the value derived from the API settings when it was unset.
func rpResolveString(configured types.String, derived string) types.String {
	if !configured.IsNull() && !configured.IsUnknown() {
		return configured
	}

	return types.StringValue(derived)
}

// rpQuotaString renders a package's quota settings as the string form the API
// accepts. A null quota (or an explicit unlimited flag) reads as "unlimited".
func rpQuotaString(s PackageSettings) string {
	if s.QuotaUnlimited || s.QuotaGB == nil {
		return "unlimited"
	}

	return strconv.FormatFloat(*s.QuotaGB, 'f', -1, 64)
}

// rpCountString renders a nullable count limit as its string form, treating a
// null (unlimited) limit as "unlimited".
func rpCountString(p *int64) string {
	if p == nil {
		return "unlimited"
	}

	return strconv.FormatInt(*p, 10)
}

// rpFloat64Value maps a nullable float to a framework Float64.
func rpFloat64Value(p *float64) types.Float64 {
	if p == nil {
		return types.Float64Null()
	}

	return types.Float64Value(*p)
}

// rpInt64Value maps a nullable int to a framework Int64.
func rpInt64Value(p *int64) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}

	return types.Int64Value(*p)
}
