package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure EmailQuotaDataSource satisfies the framework interfaces.
var (
	_ datasource.DataSource              = &EmailQuotaDataSource{}
	_ datasource.DataSourceWithConfigure = &EmailQuotaDataSource{}
)

// NewEmailQuotaDataSource returns a new mxroute_email_quota data source.
func NewEmailQuotaDataSource() datasource.DataSource {
	return &EmailQuotaDataSource{}
}

// EmailQuotaDataSource reads per-mailbox usage from the MXroute account.
type EmailQuotaDataSource struct {
	client *Client
}

// EmailQuotaDataSourceModel maps the mxroute_email_quota data source schema to
// Go values.
type EmailQuotaDataSourceModel struct {
	Username types.String `tfsdk:"username"`
	Accounts types.List   `tfsdk:"accounts"`
	ID       types.String `tfsdk:"id"`
}

// emailQuotaAccountModel maps one accounts element to Go values.
type emailQuotaAccountModel struct {
	EmailAddress types.String `tfsdk:"email_address"`
	SizeBytes    types.Int64  `tfsdk:"size_bytes"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

// emailQuotaAccountAttrTypes is the attribute type map for one accounts
// element.
func emailQuotaAccountAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"email_address": types.StringType,
		"size_bytes":    types.Int64Type,
		"updated_at":    types.StringType,
	}
}

func (d *EmailQuotaDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_email_quota"
}

func (d *EmailQuotaDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches per-mailbox storage usage from the MXroute account.",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				MarkdownDescription: "The account username the mailbox usage belongs to.",
				Computed:            true,
			},
			"accounts": schema.ListNestedAttribute{
				MarkdownDescription: "Per-mailbox storage usage.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"email_address": schema.StringAttribute{
							MarkdownDescription: "The mailbox email address.",
							Computed:            true,
						},
						"size_bytes": schema.Int64Attribute{
							MarkdownDescription: "Storage used by the mailbox, in bytes.",
							Computed:            true,
						},
						"updated_at": schema.StringAttribute{
							MarkdownDescription: "When the mailbox usage was last computed.",
							Computed:            true,
						},
					},
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Data source identifier — the account username.",
				Computed:            true,
			},
		},
	}
}

func (d *EmailQuotaDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

func (d *EmailQuotaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config EmailQuotaDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// XXX: ENVELOPE-UNVERIFIED — like GET /quota, the OpenAPI spec documents
	// GET /quota/email's 200 body as the EmailQuota object directly, which may
	// mean /quota/email is UNENVELOPED (no {success, data} wrapper). client.Do
	// always unwraps the "data" field, so if /quota/email is in fact
	// unenveloped, Do will read an empty/absent "data" and this decode will
	// silently yield a zero-valued EmailQuota. This code assumes the STANDARD
	// envelope for now (map via the shared EmailQuota model). The enveloping
	// MUST be verified against the live account (the acceptance test below is
	// the check): if the response comes back unenveloped, the fix is a
	// follow-up client change, not a change here.
	var api EmailQuota

	if err := d.client.Do(ctx, http.MethodGet, "/quota/email", nil, &api); err != nil {
		resp.Diagnostics.AddError("Error reading email quota", err.Error())

		return
	}

	accountModels := make([]emailQuotaAccountModel, 0, len(api.Accounts))

	for _, account := range api.Accounts {
		accountModels = append(accountModels, emailQuotaAccountModel{
			EmailAddress: types.StringValue(account.EmailAddress),
			SizeBytes:    types.Int64Value(account.SizeBytes),
			UpdatedAt:    types.StringValue(account.UpdatedAt),
		})
	}

	accounts, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: emailQuotaAccountAttrTypes()}, accountModels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := EmailQuotaDataSourceModel{
		Username: types.StringValue(api.Username),
		Accounts: accounts,
		ID:       types.StringValue(api.Username),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
