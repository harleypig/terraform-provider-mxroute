package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure EmailAccountResource satisfies the framework interfaces.
var (
	_ resource.Resource                = &EmailAccountResource{}
	_ resource.ResourceWithConfigure   = &EmailAccountResource{}
	_ resource.ResourceWithImportState = &EmailAccountResource{}
)

// NewEmailAccountResource returns a new mxroute_email_account resource.
func NewEmailAccountResource() resource.Resource {
	return &EmailAccountResource{}
}

// EmailAccountResource manages a mailbox on a domain hosted at MXroute.
type EmailAccountResource struct {
	client *Client
}

// EmailAccountResourceModel maps the mxroute_email_account schema to Go
// values. The mailbox password is a write-only attribute and is therefore
// never present on this model when it is written to state — PasswordWO is
// always null in state.
type EmailAccountResourceModel struct {
	Domain            types.String  `tfsdk:"domain"`
	Username          types.String  `tfsdk:"username"`
	PasswordWO        types.String  `tfsdk:"password_wo"`
	PasswordWOVersion types.Int64   `tfsdk:"password_wo_version"`
	Quota             types.Int64   `tfsdk:"quota"`
	Limit             types.Int64   `tfsdk:"limit"`
	Email             types.String  `tfsdk:"email"`
	Usage             types.Float64 `tfsdk:"usage"`
	Sent              types.Int64   `tfsdk:"sent"`
	Suspended         types.Bool    `tfsdk:"suspended"`
	ID                types.String  `tfsdk:"id"`
}

// createEmailAccountRequest is the POST email-accounts body.
type createEmailAccountRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Quota    *int64 `json:"quota,omitempty"`
}

// updateEmailAccountRequest is the PATCH email-account body; every field is
// optional so only the changed attributes are sent.
type updateEmailAccountRequest struct {
	Password *string `json:"password,omitempty"`
	Quota    *int64  `json:"quota,omitempty"`
	Limit    *int64  `json:"limit,omitempty"`
}

func (r *EmailAccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_email_account"
}

func (r *EmailAccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an email account (mailbox) on a domain hosted at MXroute. The `domain` and `username` identify the mailbox and cannot be changed in place, so changing either replaces the resource.",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain the mailbox belongs to (e.g. `example.com`). Changing this replaces the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The local part of the address (the name before the `@`). Changing this replaces the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password_wo": schema.StringAttribute{
				MarkdownDescription: "The mailbox password. This is a write-only attribute: it is sent to the API but never stored in Terraform state. Bump `password_wo_version` to change it on an existing mailbox.",
				Required:            true,
				WriteOnly:           true,
			},
			"password_wo_version": schema.Int64Attribute{
				MarkdownDescription: "Version trigger for `password_wo`. Because a write-only value cannot be diffed, increment this whenever `password_wo` changes so the new password is sent on update.",
				Optional:            true,
			},
			"quota": schema.Int64Attribute{
				MarkdownDescription: "Mailbox storage quota in megabytes. Computed from the server when not set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"limit": schema.Int64Attribute{
				MarkdownDescription: "Daily outbound send limit. Computed from the server when not set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The full email address (`username@domain`).",
				Computed:            true,
			},
			"usage": schema.Float64Attribute{
				MarkdownDescription: "Current mailbox storage usage in megabytes.",
				Computed:            true,
			},
			"sent": schema.Int64Attribute{
				MarkdownDescription: "Number of messages sent in the current window.",
				Computed:            true,
			},
			"suspended": schema.BoolAttribute{
				MarkdownDescription: "Whether the mailbox is suspended.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier — `domain/username`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *EmailAccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *EmailAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EmailAccountResourceModel

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

	domain := plan.Domain.ValueString()
	username := plan.Username.ValueString()

	body := createEmailAccountRequest{
		Username: username,
		Password: password.ValueString(),
		Quota:    int64PtrFromValue(plan.Quota),
	}

	if err := r.client.Do(ctx, http.MethodPost, "/domains/"+domain+"/email-accounts", body, nil); err != nil {
		resp.Diagnostics.AddError("Error creating email account", err.Error())

		return
	}

	// The create response carries no body; read the mailbox back to populate
	// the computed attributes.
	api, err := r.fetchEmailAccount(ctx, domain, username)
	if err != nil {
		resp.Diagnostics.AddError("Error reading email account after create", err.Error())

		return
	}

	if api == nil {
		resp.Diagnostics.AddError("Error reading email account after create", fmt.Sprintf("email account %q on %q was not found immediately after creation", username, domain))

		return
	}

	state := emailAccountStateFromAPI(api, domain, username, types.Int64Value(api.Quota), plan.PasswordWOVersion)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EmailAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EmailAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := state.Domain.ValueString()
	username := state.Username.ValueString()

	api, err := r.fetchEmailAccount(ctx, domain, username)
	if err != nil {
		resp.Diagnostics.AddError("Error reading email account", err.Error())

		return
	}

	if api == nil {
		resp.State.RemoveResource(ctx)

		return
	}

	newState := emailAccountStateFromAPI(api, domain, username, types.Int64Value(api.Quota), state.PasswordWOVersion)

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *EmailAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state EmailAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := plan.Domain.ValueString()
	username := plan.Username.ValueString()

	var body updateEmailAccountRequest

	// The password is only sent when its version trigger changed, and even
	// then it is read from config — never from plan or state.
	if !plan.PasswordWOVersion.Equal(state.PasswordWOVersion) {
		var password types.String

		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("password_wo"), &password)...)
		if resp.Diagnostics.HasError() {
			return
		}

		value := password.ValueString()
		body.Password = &value
	}

	if !plan.Quota.Equal(state.Quota) {
		body.Quota = int64PtrFromValue(plan.Quota)
	}

	if !plan.Limit.Equal(state.Limit) {
		body.Limit = int64PtrFromValue(plan.Limit)
	}

	if err := r.client.Do(ctx, http.MethodPatch, "/domains/"+domain+"/email-accounts/"+username, body, nil); err != nil {
		resp.Diagnostics.AddError("Error updating email account", err.Error())

		return
	}

	// Read the mailbox back to refresh the computed attributes.
	api, err := r.fetchEmailAccount(ctx, domain, username)
	if err != nil {
		resp.Diagnostics.AddError("Error reading email account after update", err.Error())

		return
	}

	if api == nil {
		resp.Diagnostics.AddError("Error reading email account after update", fmt.Sprintf("email account %q on %q was not found after update", username, domain))

		return
	}

	newState := emailAccountStateFromAPI(api, domain, username, types.Int64Value(api.Quota), plan.PasswordWOVersion)

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *EmailAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EmailAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := "/domains/" + state.Domain.ValueString() + "/email-accounts/" + state.Username.ValueString()

	// A mailbox already gone is a successful delete.
	if err := r.client.Do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting email account", err.Error())

		return
	}
}

func (r *EmailAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domain, username, found := strings.Cut(req.ID, "/")
	if !found || domain == "" || username == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"domain/username\", got: %q", req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), domain)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), username)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// fetchEmailAccount GETs a single mailbox, returning (nil, nil) when it does
// not exist.
func (r *EmailAccountResource) fetchEmailAccount(ctx context.Context, domain, username string) (*EmailAccount, error) {
	var api EmailAccount

	if err := r.client.Do(ctx, http.MethodGet, "/domains/"+domain+"/email-accounts/"+username, nil, &api); err != nil {
		if IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return &api, nil
}

// emailAccountStateFromAPI builds the state model from an API mailbox. The
// identity (domain, username), the write-only password's version trigger, and
// the config-supplied quota come from the caller so that non-computed
// attributes stay consistent with the plan; the remaining attributes come from
// the API. The password itself is always null in state.
func emailAccountStateFromAPI(api *EmailAccount, domain, username string, quota, passwordWOVersion types.Int64) EmailAccountResourceModel {
	return EmailAccountResourceModel{
		Domain:            types.StringValue(domain),
		Username:          types.StringValue(username),
		PasswordWO:        types.StringNull(),
		PasswordWOVersion: passwordWOVersion,
		Quota:             quota,
		Limit:             types.Int64Value(api.Limit),
		Email:             types.StringValue(api.Email),
		Usage:             types.Float64Value(api.Usage),
		Sent:              types.Int64Value(api.Sent),
		Suspended:         types.BoolValue(api.Suspended),
		ID:                types.StringValue(domain + "/" + username),
	}
}

// int64PtrFromValue returns a pointer to the value of v, or nil when v is null
// or unknown — the shape the optional API request fields expect.
func int64PtrFromValue(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}

	value := v.ValueInt64()

	return &value
}
