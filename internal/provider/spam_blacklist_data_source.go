package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure SpamBlacklistDataSource satisfies the framework interfaces.
var (
	_ datasource.DataSource              = &SpamBlacklistDataSource{}
	_ datasource.DataSourceWithConfigure = &SpamBlacklistDataSource{}
)

// NewSpamBlacklistDataSource returns a new mxroute_spam_blacklist data source.
func NewSpamBlacklistDataSource() datasource.DataSource {
	return &SpamBlacklistDataSource{}
}

// SpamBlacklistDataSource lists a domain's spam blacklist entries.
type SpamBlacklistDataSource struct {
	client *Client
}

// SpamBlacklistDataSourceModel maps the mxroute_spam_blacklist schema to Go
// values.
type SpamBlacklistDataSourceModel struct {
	Domain  types.String `tfsdk:"domain"`
	Entries types.List   `tfsdk:"entries"`
	ID      types.String `tfsdk:"id"`
}

func (d *SpamBlacklistDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_spam_blacklist"
}

func (d *SpamBlacklistDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists a domain's spam blacklist — the sender addresses and patterns always rejected. For one entry, use the `mxroute_spam_blacklist_entry` resource.",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain whose spam blacklist to list.",
				Required:            true,
			},
			"entries": schema.ListAttribute{
				MarkdownDescription: "The blacklisted sender addresses and patterns.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"id": dataSourceIDAttribute("Data source identifier — the domain name."),
		},
	}
}

func (d *SpamBlacklistDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

func (d *SpamBlacklistDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config SpamBlacklistDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := config.Domain.ValueString()

	var api []string

	if err := d.client.Do(ctx, http.MethodGet, "/domains/"+pathSeg(domain)+"/spam/blacklist", nil, &api); err != nil {
		resp.Diagnostics.AddError("Error listing spam blacklist", err.Error())

		return
	}

	entries, diags := types.ListValueFrom(ctx, types.StringType, api)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := SpamBlacklistDataSourceModel{
		Domain:  config.Domain,
		Entries: entries,
		ID:      types.StringValue(domain),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
