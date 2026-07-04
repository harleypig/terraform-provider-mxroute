package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure EmailAccountsDataSource satisfies the framework interfaces.
var (
	_ datasource.DataSource              = &EmailAccountsDataSource{}
	_ datasource.DataSourceWithConfigure = &EmailAccountsDataSource{}
)

// NewEmailAccountsDataSource returns a new mxroute_email_accounts data source.
func NewEmailAccountsDataSource() datasource.DataSource {
	return &EmailAccountsDataSource{}
}

// EmailAccountsDataSource lists the mailboxes on a domain.
type EmailAccountsDataSource struct {
	client *Client
}

// EmailAccountsDataSourceModel maps the mxroute_email_accounts schema to Go
// values.
type EmailAccountsDataSourceModel struct {
	Domain   types.String `tfsdk:"domain"`
	Accounts types.List   `tfsdk:"accounts"`
	ID       types.String `tfsdk:"id"`
}

// emailAccountsAccountModel maps one accounts element to Go values.
type emailAccountsAccountModel struct {
	Username  types.String  `tfsdk:"username"`
	Email     types.String  `tfsdk:"email"`
	Quota     types.Int64   `tfsdk:"quota"`
	Usage     types.Float64 `tfsdk:"usage"`
	Limit     types.Int64   `tfsdk:"limit"`
	Sent      types.Int64   `tfsdk:"sent"`
	Suspended types.Bool    `tfsdk:"suspended"`
}

// emailAccountsAccountAttrTypes is the attribute type map for one accounts
// element.
func emailAccountsAccountAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"username":  types.StringType,
		"email":     types.StringType,
		"quota":     types.Int64Type,
		"usage":     types.Float64Type,
		"limit":     types.Int64Type,
		"sent":      types.Int64Type,
		"suspended": types.BoolType,
	}
}

func (d *EmailAccountsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_email_accounts"
}

func (d *EmailAccountsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the mailboxes on a domain. For one mailbox, use the `mxroute_email_account` resource.",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain whose mailboxes to list.",
				Required:            true,
			},
			"accounts": schema.ListNestedAttribute{
				MarkdownDescription: "The mailboxes on the domain.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"username": schema.StringAttribute{
							MarkdownDescription: "The local part of the mailbox address.",
							Computed:            true,
						},
						"email": schema.StringAttribute{
							MarkdownDescription: "The full email address.",
							Computed:            true,
						},
						"quota": schema.Int64Attribute{
							MarkdownDescription: "Storage quota in megabytes (0 = unlimited).",
							Computed:            true,
						},
						"usage": schema.Float64Attribute{
							MarkdownDescription: "Storage currently used, in megabytes.",
							Computed:            true,
						},
						"limit": schema.Int64Attribute{
							MarkdownDescription: "Daily outbound send limit.",
							Computed:            true,
						},
						"sent": schema.Int64Attribute{
							MarkdownDescription: "Messages sent today.",
							Computed:            true,
						},
						"suspended": schema.BoolAttribute{
							MarkdownDescription: "Whether the mailbox is suspended.",
							Computed:            true,
						},
					},
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Data source identifier — the domain name.",
				Computed:            true,
			},
		},
	}
}

func (d *EmailAccountsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *EmailAccountsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config EmailAccountsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := config.Domain.ValueString()

	var api []EmailAccount

	if err := d.client.Do(ctx, http.MethodGet, "/domains/"+domain+"/email-accounts", nil, &api); err != nil {
		resp.Diagnostics.AddError("Error listing email accounts", err.Error())

		return
	}

	accountModels := make([]emailAccountsAccountModel, 0, len(api))

	for _, account := range api {
		accountModels = append(accountModels, emailAccountsAccountModel{
			Username:  types.StringValue(account.Username),
			Email:     types.StringValue(account.Email),
			Quota:     types.Int64Value(account.Quota),
			Usage:     types.Float64Value(account.Usage),
			Limit:     types.Int64Value(account.Limit),
			Sent:      types.Int64Value(account.Sent),
			Suspended: types.BoolValue(account.Suspended),
		})
	}

	accounts, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: emailAccountsAccountAttrTypes()}, accountModels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := EmailAccountsDataSourceModel{
		Domain:   config.Domain,
		Accounts: accounts,
		ID:       types.StringValue(domain),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
