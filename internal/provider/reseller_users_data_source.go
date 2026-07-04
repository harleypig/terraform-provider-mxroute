package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ResellerUsersDataSource satisfies the framework interfaces.
var (
	_ datasource.DataSource              = &ResellerUsersDataSource{}
	_ datasource.DataSourceWithConfigure = &ResellerUsersDataSource{}
)

// NewResellerUsersDataSource returns a new mxroute_reseller_users data source.
func NewResellerUsersDataSource() datasource.DataSource {
	return &ResellerUsersDataSource{}
}

// ResellerUsersDataSource lists reseller usernames. It is only usable on a
// reseller account.
type ResellerUsersDataSource struct {
	client *Client
}

// ResellerUsersDataSourceModel maps the mxroute_reseller_users schema to Go
// values.
type ResellerUsersDataSourceModel struct {
	Usernames types.List   `tfsdk:"usernames"`
	ID        types.String `tfsdk:"id"`
}

func (d *ResellerUsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reseller_users"
}

func (d *ResellerUsersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists reseller usernames on the MXroute account. Requires a reseller account. For one user's details, use the `mxroute_reseller_user` resource.",
		Attributes: map[string]schema.Attribute{
			"usernames": schema.ListAttribute{
				MarkdownDescription: "The usernames of all reseller-managed users.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Data source identifier — a fixed value for this account-wide list.",
				Computed:            true,
			},
		},
	}
}

func (d *ResellerUsersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ResellerUsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var names []string

	if err := d.client.Do(ctx, http.MethodGet, "/reseller/users", nil, &names); err != nil {
		resp.Diagnostics.AddError("Error listing reseller users", err.Error())

		return
	}

	list, diags := types.ListValueFrom(ctx, types.StringType, names)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := ResellerUsersDataSourceModel{
		Usernames: list,
		ID:        types.StringValue("reseller_users"),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
