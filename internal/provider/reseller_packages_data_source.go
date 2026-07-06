package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ResellerPackagesDataSource satisfies the framework interfaces.
var (
	_ datasource.DataSource              = &ResellerPackagesDataSource{}
	_ datasource.DataSourceWithConfigure = &ResellerPackagesDataSource{}
)

// NewResellerPackagesDataSource returns a new mxroute_reseller_packages data
// source.
func NewResellerPackagesDataSource() datasource.DataSource {
	return &ResellerPackagesDataSource{}
}

// ResellerPackagesDataSource lists reseller package names. It is only usable
// on a reseller account.
type ResellerPackagesDataSource struct {
	client *Client
}

// ResellerPackagesDataSourceModel maps the mxroute_reseller_packages schema to
// Go values.
type ResellerPackagesDataSourceModel struct {
	Names types.List   `tfsdk:"names"`
	ID    types.String `tfsdk:"id"`
}

func (d *ResellerPackagesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reseller_packages"
}

func (d *ResellerPackagesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists reseller package names on the MXroute account. Requires a reseller account. For one package's settings, use the `mxroute_reseller_package` resource.",
		Attributes: map[string]schema.Attribute{
			"names": schema.ListAttribute{
				MarkdownDescription: "The names of all reseller packages.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"id": dataSourceIDAttribute("Data source identifier — a fixed value for this account-wide list."),
		},
	}
}

func (d *ResellerPackagesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

func (d *ResellerPackagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var names []string

	if err := d.client.Do(ctx, http.MethodGet, "/reseller/packages", nil, &names); err != nil {
		resp.Diagnostics.AddError("Error listing reseller packages", err.Error())

		return
	}

	list, diags := types.ListValueFrom(ctx, types.StringType, names)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := ResellerPackagesDataSourceModel{
		Names: list,
		ID:    types.StringValue("reseller_packages"),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
