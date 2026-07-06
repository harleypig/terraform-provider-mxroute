package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure SpamSettingsResource satisfies the framework interfaces.
var (
	_ resource.Resource                = &SpamSettingsResource{}
	_ resource.ResourceWithConfigure   = &SpamSettingsResource{}
	_ resource.ResourceWithImportState = &SpamSettingsResource{}
)

// NewSpamSettingsResource returns a new mxroute_spam_settings resource.
func NewSpamSettingsResource() resource.Resource {
	return &SpamSettingsResource{}
}

// SpamSettingsResource manages a domain's spam configuration. It is a
// per-domain singleton — the domain already owns the settings, so this
// resource configures them rather than creating a separate object.
type SpamSettingsResource struct {
	client *Client
}

// SpamSettingsResourceModel maps the mxroute_spam_settings schema to Go
// values.
type SpamSettingsResourceModel struct {
	Domain    types.String `tfsdk:"domain"`
	HighScore types.Int64  `tfsdk:"high_score"`
	ID        types.String `tfsdk:"id"`
}

// spamSettingsRequest is the PATCH /domains/{domain}/spam/settings body.
type spamSettingsRequest struct {
	HighScore int64 `json:"high_score"`
}

func (r *SpamSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_spam_settings"
}

func (r *SpamSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a domain's spam configuration on the MXroute account. This is a per-domain singleton; there is no reset endpoint, so destroying the resource only drops it from Terraform state and leaves the domain's spam settings untouched.",
		Attributes: map[string]schema.Attribute{
			"domain": requiredReplaceString("The domain whose spam settings are managed (e.g. `example.com`). Changing this replaces the resource."),
			"high_score": schema.Int64Attribute{
				MarkdownDescription: "The spam score at or above which a message is auto-deleted, from 1 to 50.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 50),
				},
			},
			"id": computedIDAttribute("Resource identifier — the domain name."),
		},
	}
}

func (r *SpamSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResourceClient(req, resp); client != nil {
		r.client = client
	}
}

func (r *SpamSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SpamSettingsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, ok := r.apply(ctx, plan, &resp.Diagnostics)
	if !ok {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// apply PATCHes plan's spam settings and reads them back into the returned
// state — the shared body of Create and Update. It reports false (after adding
// a diagnostic) when the PATCH or read-back fails.
func (r *SpamSettingsResource) apply(ctx context.Context, plan SpamSettingsResourceModel, diags *diag.Diagnostics) (SpamSettingsResourceModel, bool) {
	domain := plan.Domain.ValueString()

	body := spamSettingsRequest{HighScore: plan.HighScore.ValueInt64()}

	if err := r.client.Do(ctx, http.MethodPatch, "/domains/"+domain+"/spam/settings", body, nil); err != nil {
		diags.AddError("Error applying spam settings", err.Error())

		return plan, false
	}

	api, err := r.fetchSpamSettings(ctx, domain)
	if err != nil {
		diags.AddError("Error reading spam settings after apply", err.Error())

		return plan, false
	}

	if api == nil {
		diags.AddError("Error reading spam settings after apply", fmt.Sprintf("spam settings for domain %q were not found after being applied", domain))

		return plan, false
	}

	return spamSettingsModelFromAPI(domain, api), true
}

func (r *SpamSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SpamSettingsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := state.Domain.ValueString()

	api, err := r.fetchSpamSettings(ctx, domain)
	if err != nil {
		resp.Diagnostics.AddError("Error reading spam settings", err.Error())

		return
	}

	if api == nil {
		resp.State.RemoveResource(ctx)

		return
	}

	newState := spamSettingsModelFromAPI(domain, api)

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *SpamSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SpamSettingsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, ok := r.apply(ctx, plan, &resp.Diagnostics)
	if !ok {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete drops the resource from state only. MXroute exposes no reset
// endpoint for spam settings, so the domain's configuration is left as-is.
func (r *SpamSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Nothing to do — removing from state is the framework's default once
	// this returns without error.
}

func (r *SpamSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importSingleKey(ctx, req, resp, "domain")
}

// fetchSpamSettings GETs a domain's spam settings, returning (nil, nil) when
// they do not exist.
func (r *SpamSettingsResource) fetchSpamSettings(ctx context.Context, domain string) (*SpamSettings, error) {
	return fetchOne[SpamSettings](ctx, r.client, "/domains/"+domain+"/spam/settings")
}

// spamSettingsModelFromAPI maps API spam settings onto the Terraform state
// model. The id is the domain, which is the singleton's identity.
func spamSettingsModelFromAPI(domain string, api *SpamSettings) SpamSettingsResourceModel {
	return SpamSettingsResourceModel{
		Domain:    types.StringValue(domain),
		HighScore: types.Int64Value(api.HighScore),
		ID:        types.StringValue(domain),
	}
}
