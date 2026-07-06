package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ResellerUserDataSource satisfies the framework interfaces.
var (
	_ datasource.DataSource              = &ResellerUserDataSource{}
	_ datasource.DataSourceWithConfigure = &ResellerUserDataSource{}
)

// NewResellerUserDataSource returns a new mxroute_reseller_user data source.
func NewResellerUserDataSource() datasource.DataSource {
	return &ResellerUserDataSource{}
}

// ResellerUserDataSource fetches a single reseller user by username.
type ResellerUserDataSource struct {
	client *Client
}

// ResellerUserDataSourceModel maps the mxroute_reseller_user schema to Go
// values.
type ResellerUserDataSourceModel struct {
	Username       types.String  `tfsdk:"username"`
	Email          types.String  `tfsdk:"email"`
	Domain         types.String  `tfsdk:"domain"`
	Package        types.String  `tfsdk:"package"`
	Suspended      types.Bool    `tfsdk:"suspended"`
	QuotaLimit     types.Int64   `tfsdk:"quota_limit"`
	QuotaUsed      types.Float64 `tfsdk:"quota_used"`
	QuotaUnlimited types.Bool    `tfsdk:"quota_unlimited"`
	ID             types.String  `tfsdk:"id"`
}

func (d *ResellerUserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reseller_user"
}

func (d *ResellerUserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a single reseller user by username. Requires reseller API access. For all users, use `mxroute_reseller_users`.",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				MarkdownDescription: "The username to look up.",
				Required:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The user's contact email address.",
				Computed:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The primary domain associated with the user.",
				Computed:            true,
			},
			"package": schema.StringAttribute{
				MarkdownDescription: "The reseller package assigned to the user.",
				Computed:            true,
			},
			"suspended": schema.BoolAttribute{
				MarkdownDescription: "Whether the user is suspended.",
				Computed:            true,
			},
			"quota_limit": schema.Int64Attribute{
				MarkdownDescription: "The user's storage quota limit in megabytes; null when unlimited.",
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
			"id": dataSourceIDAttribute("Data source identifier — the username."),
		},
	}
}

func (d *ResellerUserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

func (d *ResellerUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ResellerUserDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	username := config.Username.ValueString()

	var api ResellerUser

	if err := d.client.Do(ctx, http.MethodGet, "/reseller/users/"+pathSeg(username), nil, &api); err != nil {
		resp.Diagnostics.AddError("Error reading reseller user", err.Error())

		return
	}

	state := ResellerUserDataSourceModel{
		Username:       types.StringValue(api.Username),
		Email:          types.StringValue(api.Email),
		Domain:         types.StringValue(api.Domain),
		Package:        types.StringValue(api.Package),
		Suspended:      types.BoolValue(api.Suspended),
		QuotaLimit:     rpInt64Value(api.Quota.Limit),
		QuotaUsed:      types.Float64Value(api.Quota.Used),
		QuotaUnlimited: types.BoolValue(api.Quota.Unlimited),
		ID:             types.StringValue(api.Username),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
