package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure SpamWhitelistDataSource satisfies the framework interfaces.
var (
	_ datasource.DataSource              = &SpamWhitelistDataSource{}
	_ datasource.DataSourceWithConfigure = &SpamWhitelistDataSource{}
)

// NewSpamWhitelistDataSource returns a new mxroute_spam_whitelist data source.
func NewSpamWhitelistDataSource() datasource.DataSource {
	return &SpamWhitelistDataSource{}
}

// SpamWhitelistDataSource lists a domain's spam whitelist entries.
type SpamWhitelistDataSource struct {
	client *Client
}

// SpamWhitelistDataSourceModel maps the mxroute_spam_whitelist schema to Go
// values.
type SpamWhitelistDataSourceModel struct {
	Domain  types.String `tfsdk:"domain"`
	Entries types.List   `tfsdk:"entries"`
	ID      types.String `tfsdk:"id"`
}

func (d *SpamWhitelistDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_spam_whitelist"
}

func (d *SpamWhitelistDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists a domain's spam whitelist — the sender addresses and patterns always accepted. For one entry, use the `mxroute_spam_whitelist_entry` resource.",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain whose spam whitelist to list.",
				Required:            true,
			},
			"entries": schema.ListAttribute{
				MarkdownDescription: "The whitelisted sender addresses and patterns.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"id": dataSourceIDAttribute("Data source identifier — the domain name."),
		},
	}
}

func (d *SpamWhitelistDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

func (d *SpamWhitelistDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config SpamWhitelistDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := config.Domain.ValueString()

	var api []string

	if err := d.client.Do(ctx, http.MethodGet, "/domains/"+pathSeg(domain)+"/spam/whitelist", nil, &api); err != nil {
		resp.Diagnostics.AddError("Error listing spam whitelist", err.Error())

		return
	}

	entries, diags := types.ListValueFrom(ctx, types.StringType, api)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := SpamWhitelistDataSourceModel{
		Domain:  config.Domain,
		Entries: entries,
		ID:      types.StringValue(domain),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
