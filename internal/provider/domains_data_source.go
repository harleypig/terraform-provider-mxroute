package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure DomainsDataSource satisfies the framework interfaces.
var (
	_ datasource.DataSource              = &DomainsDataSource{}
	_ datasource.DataSourceWithConfigure = &DomainsDataSource{}
)

// NewDomainsDataSource returns a new mxroute_domains data source.
func NewDomainsDataSource() datasource.DataSource {
	return &DomainsDataSource{}
}

// DomainsDataSource lists every mail domain on the MXroute account.
type DomainsDataSource struct {
	client *Client
}

// DomainsDataSourceModel maps the mxroute_domains data source schema to Go
// values.
type DomainsDataSourceModel struct {
	Domains types.List   `tfsdk:"domains"`
	ID      types.String `tfsdk:"id"`
}

func (d *DomainsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domains"
}

func (d *DomainsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists every mail domain on the MXroute account. For one domain's details, use the `mxroute_domain` data source.",
		Attributes: map[string]schema.Attribute{
			"domains": schema.ListAttribute{
				MarkdownDescription: "The names of all domains on the account.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"id": dataSourceIDAttribute("Data source identifier — a fixed value for this account-wide list."),
		},
	}
}

func (d *DomainsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

func (d *DomainsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var names []string

	if err := d.client.Do(ctx, http.MethodGet, "/domains", nil, &names); err != nil {
		resp.Diagnostics.AddError("Error listing domains", err.Error())

		return
	}

	domains, diags := types.ListValueFrom(ctx, types.StringType, names)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := DomainsDataSourceModel{
		Domains: domains,
		ID:      types.StringValue("domains"),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
