package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure DNSDataSource satisfies the framework interfaces.
var (
	_ datasource.DataSource              = &DNSDataSource{}
	_ datasource.DataSourceWithConfigure = &DNSDataSource{}
)

// NewDNSDataSource returns a new mxroute_dns data source.
func NewDNSDataSource() datasource.DataSource {
	return &DNSDataSource{}
}

// DNSDataSource reads the DNS records MXroute publishes for a mail domain.
type DNSDataSource struct {
	client *Client
}

// DNSDataSourceModel maps the mxroute_dns data source schema to Go values.
// The nested objects are held as framework types so a null dkim or
// verification stays null rather than materializing an empty object.
type DNSDataSourceModel struct {
	Domain       types.String `tfsdk:"domain"`
	MXRecords    types.List   `tfsdk:"mx_records"`
	SPF          types.Object `tfsdk:"spf"`
	DKIM         types.Object `tfsdk:"dkim"`
	Verification types.Object `tfsdk:"verification"`
	ID           types.String `tfsdk:"id"`
}

// dnsMXRecordModel maps a single mx_records entry to Go values.
type dnsMXRecordModel struct {
	Priority    types.Int64  `tfsdk:"priority"`
	Hostname    types.String `tfsdk:"hostname"`
	Description types.String `tfsdk:"description"`
}

// dnsRecordModel maps a {type, name, value} DNS record to Go values.
type dnsRecordModel struct {
	Type  types.String `tfsdk:"type"`
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

// dnsVerificationModel maps the verification record to Go values.
type dnsVerificationModel struct {
	Type        types.String `tfsdk:"type"`
	Name        types.String `tfsdk:"name"`
	Value       types.String `tfsdk:"value"`
	Description types.String `tfsdk:"description"`
}

// dnsMXRecordAttrTypes is the attribute type map for one mx_records entry.
func dnsMXRecordAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"priority":    types.Int64Type,
		"hostname":    types.StringType,
		"description": types.StringType,
	}
}

// dnsRecordAttrTypes is the attribute type map for a {type, name, value}
// record (spf and dkim).
func dnsRecordAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":  types.StringType,
		"name":  types.StringType,
		"value": types.StringType,
	}
}

// dnsVerificationAttrTypes is the attribute type map for the verification
// record.
func dnsVerificationAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":        types.StringType,
		"name":        types.StringType,
		"value":       types.StringType,
		"description": types.StringType,
	}
}

func (d *DNSDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns"
}

func (d *DNSDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches the DNS records MXroute publishes for a mail domain (MX, SPF, DKIM, and verification).",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain name to look up DNS records for.",
				Required:            true,
			},
			"mx_records": schema.ListNestedAttribute{
				MarkdownDescription: "The MX records for the domain, in priority order.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"priority": schema.Int64Attribute{
							MarkdownDescription: "The MX record priority (lower is preferred).",
							Computed:            true,
						},
						"hostname": schema.StringAttribute{
							MarkdownDescription: "The mail server hostname.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "A human-readable description of the record.",
							Computed:            true,
						},
					},
				},
			},
			"spf": schema.SingleNestedAttribute{
				MarkdownDescription: "The SPF record for the domain.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						MarkdownDescription: "The DNS record type.",
						Computed:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "The DNS record name.",
						Computed:            true,
					},
					"value": schema.StringAttribute{
						MarkdownDescription: "The DNS record value.",
						Computed:            true,
					},
				},
			},
			"dkim": schema.SingleNestedAttribute{
				MarkdownDescription: "The DKIM record for the domain, or null when none is published.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						MarkdownDescription: "The DNS record type.",
						Computed:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "The DNS record name.",
						Computed:            true,
					},
					"value": schema.StringAttribute{
						MarkdownDescription: "The DNS record value.",
						Computed:            true,
					},
				},
			},
			"verification": schema.SingleNestedAttribute{
				MarkdownDescription: "The domain verification record, or null when none is published.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						MarkdownDescription: "The DNS record type.",
						Computed:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "The DNS record name.",
						Computed:            true,
					},
					"value": schema.StringAttribute{
						MarkdownDescription: "The DNS record value.",
						Computed:            true,
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "A human-readable description of the record.",
						Computed:            true,
					},
				},
			},
			"id": dataSourceIDAttribute("Data source identifier — the domain name."),
		},
	}
}

func (d *DNSDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

func (d *DNSDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DNSDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := config.Domain.ValueString()

	var api DNSInfo

	if err := d.client.Do(ctx, http.MethodGet, "/domains/"+domain+"/dns", nil, &api); err != nil {
		resp.Diagnostics.AddError("Error reading domain DNS", err.Error())

		return
	}

	mxModels := make([]dnsMXRecordModel, 0, len(api.MXRecords))

	for _, record := range api.MXRecords {
		mxModels = append(mxModels, dnsMXRecordModel{
			Priority:    types.Int64Value(record.Priority),
			Hostname:    types.StringValue(record.Hostname),
			Description: types.StringValue(record.Description),
		})
	}

	mxRecords, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: dnsMXRecordAttrTypes()}, mxModels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	spf, diags := dnsRecordObject(ctx, &api.SPF)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dkim, diags := dnsRecordObject(ctx, api.DKIM)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	verification, diags := dnsVerificationObject(ctx, api.Verification)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := DNSDataSourceModel{
		Domain:       types.StringValue(domain),
		MXRecords:    mxRecords,
		SPF:          spf,
		DKIM:         dkim,
		Verification: verification,
		ID:           types.StringValue(domain),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// dnsRecordObject builds a framework object for a {type, name, value}
// record, returning a null object when the API returned null.
func dnsRecordObject(ctx context.Context, api *DNSRecord) (types.Object, diag.Diagnostics) {
	if api == nil {
		return types.ObjectNull(dnsRecordAttrTypes()), nil
	}

	return types.ObjectValueFrom(ctx, dnsRecordAttrTypes(), dnsRecordModel{
		Type:  types.StringValue(api.Type),
		Name:  types.StringValue(api.Name),
		Value: types.StringValue(api.Value),
	})
}

// dnsVerificationObject builds a framework object for the verification
// record, returning a null object when the API returned null.
func dnsVerificationObject(ctx context.Context, api *DNSVerification) (types.Object, diag.Diagnostics) {
	if api == nil {
		return types.ObjectNull(dnsVerificationAttrTypes()), nil
	}

	return types.ObjectValueFrom(ctx, dnsVerificationAttrTypes(), dnsVerificationModel{
		Type:        types.StringValue(api.Type),
		Name:        types.StringValue(api.Name),
		Value:       types.StringValue(api.Value),
		Description: types.StringValue(api.Description),
	})
}
