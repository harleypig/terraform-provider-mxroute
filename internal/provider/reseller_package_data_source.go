package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ResellerPackageDataSource satisfies the framework interfaces.
var (
	_ datasource.DataSource              = &ResellerPackageDataSource{}
	_ datasource.DataSourceWithConfigure = &ResellerPackageDataSource{}
)

// NewResellerPackageDataSource returns a new mxroute_reseller_package data
// source.
func NewResellerPackageDataSource() datasource.DataSource {
	return &ResellerPackageDataSource{}
}

// ResellerPackageDataSource fetches a single reseller package by name.
type ResellerPackageDataSource struct {
	client *Client
}

// ResellerPackageDataSourceModel maps the mxroute_reseller_package schema to Go
// values.
type ResellerPackageDataSourceModel struct {
	Name     types.String `tfsdk:"name"`
	Settings types.Object `tfsdk:"settings"`
	ID       types.String `tfsdk:"id"`
}

func (d *ResellerPackageDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reseller_package"
}

func (d *ResellerPackageDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a single reseller package by name. Requires reseller API access. For all packages, use `mxroute_reseller_packages`.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The package name to look up.",
				Required:            true,
			},
			"settings": schema.SingleNestedAttribute{
				MarkdownDescription: "The limits the package grants. A null count means unlimited.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"quota_gb": schema.Float64Attribute{
						MarkdownDescription: "Total storage quota in gigabytes; null when unlimited.",
						Computed:            true,
					},
					"quota_unlimited": schema.BoolAttribute{
						MarkdownDescription: "Whether the storage quota is unlimited.",
						Computed:            true,
					},
					"domains": schema.Int64Attribute{
						MarkdownDescription: "Maximum domains; null when unlimited.",
						Computed:            true,
					},
					"email_accounts": schema.Int64Attribute{
						MarkdownDescription: "Maximum mailboxes; null when unlimited.",
						Computed:            true,
					},
					"email_forwarders": schema.Int64Attribute{
						MarkdownDescription: "Maximum forwarders; null when unlimited.",
						Computed:            true,
					},
					"domain_pointers": schema.Int64Attribute{
						MarkdownDescription: "Maximum domain pointers; null when unlimited.",
						Computed:            true,
					},
				},
			},
			"id": dataSourceIDAttribute("Data source identifier — the package name."),
		},
	}
}

func (d *ResellerPackageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

func (d *ResellerPackageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ResellerPackageDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := config.Name.ValueString()

	var api Package

	if err := d.client.Do(ctx, http.MethodGet, "/reseller/packages/"+pathSeg(name), nil, &api); err != nil {
		resp.Diagnostics.AddError("Error reading reseller package", err.Error())

		return
	}

	settingsModel := resellerPackageSettingsModel{
		QuotaGB:         rpFloat64Value(api.Settings.QuotaGB),
		QuotaUnlimited:  types.BoolValue(api.Settings.QuotaUnlimited),
		Domains:         rpInt64Value(api.Settings.Domains),
		EmailAccounts:   rpInt64Value(api.Settings.EmailAccounts),
		EmailForwarders: rpInt64Value(api.Settings.EmailForwarders),
		DomainPointers:  rpInt64Value(api.Settings.DomainPointers),
	}

	settingsObj, diags := types.ObjectValueFrom(ctx, resellerPackageSettingsAttrTypes, settingsModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := ResellerPackageDataSourceModel{
		Name:     types.StringValue(api.Name),
		Settings: settingsObj,
		ID:       types.StringValue(api.Name),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
