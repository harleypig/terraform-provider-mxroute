package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure PointersDataSource satisfies the framework interfaces.
var (
	_ datasource.DataSource              = &PointersDataSource{}
	_ datasource.DataSourceWithConfigure = &PointersDataSource{}
)

// NewPointersDataSource returns a new mxroute_pointers data source.
func NewPointersDataSource() datasource.DataSource {
	return &PointersDataSource{}
}

// PointersDataSource lists the pointers on a domain.
type PointersDataSource struct {
	client *Client
}

// PointersDataSourceModel maps the mxroute_pointers schema to Go values.
type PointersDataSourceModel struct {
	Domain   types.String `tfsdk:"domain"`
	Pointers types.List   `tfsdk:"pointers"`
	ID       types.String `tfsdk:"id"`
}

// pointersPointerModel maps one pointers element to Go values.
type pointersPointerModel struct {
	Pointer types.String `tfsdk:"pointer"`
	Type    types.String `tfsdk:"type"`
	Target  types.String `tfsdk:"target"`
}

// pointersPointerAttrTypes is the attribute type map for one pointers element.
func pointersPointerAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"pointer": types.StringType,
		"type":    types.StringType,
		"target":  types.StringType,
	}
}

func (d *PointersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pointers"
}

func (d *PointersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the pointers (aliases and redirects) on a domain. For one pointer, use the `mxroute_pointer` resource.",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain whose pointers to list.",
				Required:            true,
			},
			"pointers": schema.ListNestedAttribute{
				MarkdownDescription: "The pointers on the domain.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"pointer": schema.StringAttribute{
							MarkdownDescription: "The pointer name that resolves to the parent domain.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The pointer type — `alias` or `redirect`.",
							Computed:            true,
						},
						"target": schema.StringAttribute{
							MarkdownDescription: "The target the pointer resolves to.",
							Computed:            true,
						},
					},
				},
			},
			"id": dataSourceIDAttribute("Data source identifier — the domain name."),
		},
	}
}

func (d *PointersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

func (d *PointersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PointersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := config.Domain.ValueString()

	var api []DomainPointer

	if err := d.client.Do(ctx, http.MethodGet, "/domains/"+pathSeg(domain)+"/pointers", nil, &api); err != nil {
		resp.Diagnostics.AddError("Error listing pointers", err.Error())

		return
	}

	pointerModels := make([]pointersPointerModel, 0, len(api))

	for _, pointer := range api {
		pointerModels = append(pointerModels, pointersPointerModel{
			Pointer: types.StringValue(pointer.Pointer),
			Type:    types.StringValue(pointer.Type),
			Target:  types.StringValue(pointer.Target),
		})
	}

	pointers, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: pointersPointerAttrTypes()}, pointerModels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := PointersDataSourceModel{
		Domain:   config.Domain,
		Pointers: pointers,
		ID:       types.StringValue(domain),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
