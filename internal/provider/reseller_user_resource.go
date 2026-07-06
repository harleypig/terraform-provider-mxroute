package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ResellerUserResource satisfies the framework interfaces.
var (
	_ resource.Resource                = &ResellerUserResource{}
	_ resource.ResourceWithConfigure   = &ResellerUserResource{}
	_ resource.ResourceWithImportState = &ResellerUserResource{}
)

// NewResellerUserResource returns a new mxroute_reseller_user resource.
func NewResellerUserResource() resource.Resource {
	return &ResellerUserResource{}
}

// ResellerUserResource manages a reseller-managed user on the MXroute
// account.
type ResellerUserResource struct {
	client *Client
}

// ResellerUserResourceModel maps the mxroute_reseller_user schema to Go
// values. The user password is a write-only attribute and is therefore never
// present on this model when it is written to state — PasswordWO is always
// null in state.
type ResellerUserResourceModel struct {
	Username          types.String  `tfsdk:"username"`
	Email             types.String  `tfsdk:"email"`
	Package           types.String  `tfsdk:"package"`
	PasswordWO        types.String  `tfsdk:"password_wo"`
	PasswordWOVersion types.Int64   `tfsdk:"password_wo_version"`
	Quota             types.String  `tfsdk:"quota"`
	Suspended         types.Bool    `tfsdk:"suspended"`
	Domain            types.String  `tfsdk:"domain"`
	QuotaUsed         types.Float64 `tfsdk:"quota_used"`
	QuotaUnlimited    types.Bool    `tfsdk:"quota_unlimited"`
	QuotaLimit        types.Int64   `tfsdk:"quota_limit"`
	ID                types.String  `tfsdk:"id"`
}

// createResellerUserRequest is the POST /reseller/users body.
type createResellerUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Package  string `json:"package"`
}

// updateResellerUserRequest is the PATCH /reseller/users/{username} body; both
// fields are optional so only the changed attributes are sent.
type updateResellerUserRequest struct {
	Quota    *string `json:"quota,omitempty"`
	Password *string `json:"password,omitempty"`
}

// resellerUserPackageRequest is the PATCH /reseller/users/{username}/package
// body.
type resellerUserPackageRequest struct {
	Package string `json:"package"`
}

func (r *ResellerUserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reseller_user"
}

func (r *ResellerUserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a reseller-managed user on the MXroute account. The `username` identifies the user and the `email` cannot be changed in place, so changing either replaces the resource.",
		Attributes: map[string]schema.Attribute{
			"username": requiredReplaceString("The reseller user's login name. Changing this replaces the resource."),
			"email":    requiredReplaceString("The reseller user's contact email address. MXroute exposes no update for it, so changing this replaces the resource."),
			"package": schema.StringAttribute{
				MarkdownDescription: "The reseller package assigned to the user. Changing this reassigns the package in place.",
				Required:            true,
			},
			"password_wo": schema.StringAttribute{
				MarkdownDescription: "The user's password. This is a write-only attribute: it is sent to the API but never stored in Terraform state. Bump `password_wo_version` to change it on an existing user.",
				Required:            true,
				WriteOnly:           true,
			},
			"password_wo_version": schema.Int64Attribute{
				MarkdownDescription: "Version trigger for `password_wo`. Because a write-only value cannot be diffed, increment this whenever `password_wo` changes so the new password is sent on update.",
				Optional:            true,
			},
			"quota": schema.StringAttribute{
				MarkdownDescription: "The user's storage quota, either a size in megabytes (e.g. `500MB`) or `unlimited`. When set, it is applied via a follow-up update.",
				Optional:            true,
			},
			"suspended": schema.BoolAttribute{
				MarkdownDescription: "Whether the user is suspended. Set it explicitly to suspend or unsuspend the user; computed from the server when not set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The primary domain associated with the reseller user.",
				Computed:            true,
			},
			"quota_used": schema.Float64Attribute{
				MarkdownDescription: "Current storage usage in megabytes.",
				Computed:            true,
			},
			"quota_unlimited": schema.BoolAttribute{
				MarkdownDescription: "Whether the user's quota is unlimited.",
				Computed:            true,
			},
			"quota_limit": schema.Int64Attribute{
				MarkdownDescription: "The user's storage quota limit in megabytes; null when the quota is unlimited.",
				Computed:            true,
			},
			"id": computedIDAttribute("Resource identifier — the username."),
		},
	}
}

func (r *ResellerUserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResourceClient(req, resp); client != nil {
		r.client = client
	}
}

func (r *ResellerUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResellerUserResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The password is write-only, so it is read from config, never from the
	// plan or state.
	var password types.String

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("password_wo"), &password)...)
	if resp.Diagnostics.HasError() {
		return
	}

	username := plan.Username.ValueString()

	body := createResellerUserRequest{
		Username: username,
		Email:    plan.Email.ValueString(),
		Password: password.ValueString(),
		Package:  plan.Package.ValueString(),
	}

	if err := r.client.Do(ctx, http.MethodPost, "/reseller/users", body, nil); err != nil {
		resp.Diagnostics.AddError("Error creating reseller user", err.Error())

		return
	}

	// Quota is not part of create; apply it with a follow-up update when set.
	if !plan.Quota.IsNull() && !plan.Quota.IsUnknown() {
		quota := plan.Quota.ValueString()

		if err := r.client.Do(ctx, http.MethodPatch, "/reseller/users/"+username, updateResellerUserRequest{Quota: &quota}, nil); err != nil {
			resp.Diagnostics.AddError("Error setting reseller user quota", err.Error())

			return
		}
	}

	// Suspend when the plan explicitly asks for it.
	if !plan.Suspended.IsNull() && !plan.Suspended.IsUnknown() && plan.Suspended.ValueBool() {
		if err := r.client.Do(ctx, http.MethodPost, "/reseller/users/"+username+"/suspend", nil, nil); err != nil {
			resp.Diagnostics.AddError("Error suspending reseller user", err.Error())

			return
		}
	}

	// The create response is partial; read the user back to populate the
	// computed attributes.
	api, err := r.fetchResellerUser(ctx, username)
	if err != nil {
		resp.Diagnostics.AddError("Error reading reseller user after create", err.Error())

		return
	}

	if api == nil {
		resp.Diagnostics.AddError("Error reading reseller user after create", fmt.Sprintf("reseller user %q was not found immediately after creation", username))

		return
	}

	state := resellerUserStateFromAPI(api, plan.Quota, plan.PasswordWOVersion)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ResellerUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResellerUserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	api, err := r.fetchResellerUser(ctx, state.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading reseller user", err.Error())

		return
	}

	if api == nil {
		resp.State.RemoveResource(ctx)

		return
	}

	newState := resellerUserStateFromAPI(api, state.Quota, state.PasswordWOVersion)

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *ResellerUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ResellerUserResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	username := plan.Username.ValueString()

	// A changed package is reassigned through the dedicated package endpoint.
	if !plan.Package.Equal(state.Package) {
		body := resellerUserPackageRequest{Package: plan.Package.ValueString()}

		if err := r.client.Do(ctx, http.MethodPatch, "/reseller/users/"+username+"/package", body, nil); err != nil {
			resp.Diagnostics.AddError("Error updating reseller user package", err.Error())

			return
		}
	}

	// The password and quota share the user PATCH endpoint; only the changed
	// fields are sent.
	var body updateResellerUserRequest

	// The password is only sent when its version trigger changed, and even then
	// it is read from config — never from plan or state.
	if !plan.PasswordWOVersion.Equal(state.PasswordWOVersion) {
		var password types.String

		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("password_wo"), &password)...)
		if resp.Diagnostics.HasError() {
			return
		}

		value := password.ValueString()
		body.Password = &value
	}

	// Quota is sent only when it changed to a concrete value; MXroute exposes no
	// way to clear it, so a change to null is left unmanaged.
	if !plan.Quota.Equal(state.Quota) && !plan.Quota.IsNull() && !plan.Quota.IsUnknown() {
		quota := plan.Quota.ValueString()
		body.Quota = &quota
	}

	if body.Password != nil || body.Quota != nil {
		if err := r.client.Do(ctx, http.MethodPatch, "/reseller/users/"+username, body, nil); err != nil {
			resp.Diagnostics.AddError("Error updating reseller user", err.Error())

			return
		}
	}

	// A changed suspended state is applied through the suspend/unsuspend
	// endpoints.
	if !plan.Suspended.IsUnknown() && !plan.Suspended.Equal(state.Suspended) {
		action := "unsuspend"
		if plan.Suspended.ValueBool() {
			action = "suspend"
		}

		if err := r.client.Do(ctx, http.MethodPost, "/reseller/users/"+username+"/"+action, nil, nil); err != nil {
			resp.Diagnostics.AddError("Error updating reseller user suspension", err.Error())

			return
		}
	}

	// Read the user back to refresh the computed attributes.
	api, err := r.fetchResellerUser(ctx, username)
	if err != nil {
		resp.Diagnostics.AddError("Error reading reseller user after update", err.Error())

		return
	}

	if api == nil {
		resp.Diagnostics.AddError("Error reading reseller user after update", fmt.Sprintf("reseller user %q was not found after update", username))

		return
	}

	newState := resellerUserStateFromAPI(api, plan.Quota, plan.PasswordWOVersion)

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *ResellerUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResellerUserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A user already gone is a successful delete.
	if err := r.client.Do(ctx, http.MethodDelete, "/reseller/users/"+state.Username.ValueString(), nil, nil); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting reseller user", err.Error())

		return
	}
}

func (r *ResellerUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// fetchResellerUser GETs a single reseller user, returning (nil, nil) when it
// does not exist.
func (r *ResellerUserResource) fetchResellerUser(ctx context.Context, username string) (*ResellerUser, error) {
	return fetchOne[ResellerUser](ctx, r.client, "/reseller/users/"+username)
}

// resellerUserStateFromAPI builds the state model from an API reseller user.
// The config-supplied quota and the write-only password's version trigger come
// from the caller so that non-computed attributes stay consistent with the
// plan; the remaining attributes come from the API. The password itself is
// always null in state.
func resellerUserStateFromAPI(api *ResellerUser, quota types.String, passwordWOVersion types.Int64) ResellerUserResourceModel {
	return ResellerUserResourceModel{
		Username:          types.StringValue(api.Username),
		Email:             types.StringValue(api.Email),
		Package:           types.StringValue(api.Package),
		PasswordWO:        types.StringNull(),
		PasswordWOVersion: passwordWOVersion,
		Quota:             quota,
		Suspended:         types.BoolValue(api.Suspended),
		Domain:            types.StringValue(api.Domain),
		QuotaUsed:         types.Float64Value(api.Quota.Used),
		QuotaUnlimited:    types.BoolValue(api.Quota.Unlimited),
		QuotaLimit:        types.Int64PointerValue(api.Quota.Limit),
		ID:                types.StringValue(api.Username),
	}
}
