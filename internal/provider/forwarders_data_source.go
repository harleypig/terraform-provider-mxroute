package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ForwardersDataSource satisfies the framework interfaces.
var (
	_ datasource.DataSource              = &ForwardersDataSource{}
	_ datasource.DataSourceWithConfigure = &ForwardersDataSource{}
)

// NewForwardersDataSource returns a new mxroute_forwarders data source.
func NewForwardersDataSource() datasource.DataSource {
	return &ForwardersDataSource{}
}

// ForwardersDataSource lists the forwarders on a domain.
type ForwardersDataSource struct {
	client *Client
}

// ForwardersDataSourceModel maps the mxroute_forwarders schema to Go values.
type ForwardersDataSourceModel struct {
	Domain     types.String `tfsdk:"domain"`
	Forwarders types.List   `tfsdk:"forwarders"`
	ID         types.String `tfsdk:"id"`
}

// forwardersForwarderModel maps one forwarders element to Go values.
type forwardersForwarderModel struct {
	Alias        types.String `tfsdk:"alias"`
	Email        types.String `tfsdk:"email"`
	Destinations types.List   `tfsdk:"destinations"`
}

// forwardersForwarderAttrTypes is the attribute type map for one forwarders
// element.
func forwardersForwarderAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"alias":        types.StringType,
		"email":        types.StringType,
		"destinations": types.ListType{ElemType: types.StringType},
	}
}

func (d *ForwardersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_forwarders"
}

func (d *ForwardersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the email forwarders (aliases) on a domain. For one forwarder, use the `mxroute_forwarder` resource.",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain whose forwarders to list.",
				Required:            true,
			},
			"forwarders": schema.ListNestedAttribute{
				MarkdownDescription: "The forwarders on the domain.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"alias": schema.StringAttribute{
							MarkdownDescription: "The local part of the forwarding address.",
							Computed:            true,
						},
						"email": schema.StringAttribute{
							MarkdownDescription: "The full forwarding address.",
							Computed:            true,
						},
						"destinations": schema.ListAttribute{
							MarkdownDescription: "The addresses mail to this alias is forwarded to.",
							ElementType:         types.StringType,
							Computed:            true,
						},
					},
				},
			},
			"id": dataSourceIDAttribute("Data source identifier — the domain name."),
		},
	}
}

func (d *ForwardersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

func (d *ForwardersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ForwardersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := config.Domain.ValueString()

	var api []Forwarder

	if err := d.client.Do(ctx, http.MethodGet, "/domains/"+pathSeg(domain)+"/forwarders", nil, &api); err != nil {
		resp.Diagnostics.AddError("Error listing forwarders", err.Error())

		return
	}

	forwarderModels := make([]forwardersForwarderModel, 0, len(api))

	for _, forwarder := range api {
		destinations, diags := types.ListValueFrom(ctx, types.StringType, forwarder.Destinations)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		forwarderModels = append(forwarderModels, forwardersForwarderModel{
			Alias:        types.StringValue(forwarder.Alias),
			Email:        types.StringValue(forwarder.Email),
			Destinations: destinations,
		})
	}

	forwarders, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: forwardersForwarderAttrTypes()}, forwarderModels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := ForwardersDataSourceModel{
		Domain:     config.Domain,
		Forwarders: forwarders,
		ID:         types.StringValue(domain),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
