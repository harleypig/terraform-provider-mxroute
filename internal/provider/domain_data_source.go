package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure DomainDataSource satisfies the framework interfaces.
var (
	_ datasource.DataSource              = &DomainDataSource{}
	_ datasource.DataSourceWithConfigure = &DomainDataSource{}
)

// NewDomainDataSource returns a new mxroute_domain data source.
func NewDomainDataSource() datasource.DataSource {
	return &DomainDataSource{}
}

// DomainDataSource reads an existing mail domain from the MXroute account.
type DomainDataSource struct {
	client *Client
}

// DomainDataSourceModel maps the mxroute_domain data source schema to Go values.
type DomainDataSourceModel struct {
	Domain      types.String `tfsdk:"domain"`
	MailHosting types.Bool   `tfsdk:"mail_hosting"`
	SSLEnabled  types.Bool   `tfsdk:"ssl_enabled"`
	Pointers    types.List   `tfsdk:"pointers"`
	ID          types.String `tfsdk:"id"`
}

func (d *DomainDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (d *DomainDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches an existing mail domain from the MXroute account.",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain name to look up.",
				Required:            true,
			},
			"mail_hosting": schema.BoolAttribute{
				MarkdownDescription: "Whether mail hosting is enabled for the domain.",
				Computed:            true,
			},
			"ssl_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether SSL is enabled for the domain. Server-managed via MXroute's AutoSSL: typically `false` immediately after a domain is created and becomes `true` asynchronously — often within ~24 hours — once the certificate is issued.",
				Computed:            true,
			},
			"pointers": schema.ListAttribute{
				MarkdownDescription: "Domain pointers (aliases) that resolve to this domain.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"id": dataSourceIDAttribute("Data source identifier — the domain name."),
		},
	}
}

func (d *DomainDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

func (d *DomainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DomainDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api Domain

	if err := d.client.Do(ctx, http.MethodGet, "/domains/"+config.Domain.ValueString(), nil, &api); err != nil {
		resp.Diagnostics.AddError("Error reading domain", err.Error())

		return
	}

	pointers, diags := types.ListValueFrom(ctx, types.StringType, api.Pointers)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := DomainDataSourceModel{
		Domain:      types.StringValue(api.Domain),
		MailHosting: types.BoolValue(api.MailHosting),
		SSLEnabled:  types.BoolValue(api.SSLEnabled),
		Pointers:    pointers,
		ID:          types.StringValue(api.Domain),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
