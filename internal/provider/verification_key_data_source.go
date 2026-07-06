package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure VerificationKeyDataSource satisfies the framework interfaces.
var (
	_ datasource.DataSource              = &VerificationKeyDataSource{}
	_ datasource.DataSourceWithConfigure = &VerificationKeyDataSource{}
)

// NewVerificationKeyDataSource returns a new mxroute_verification_key data
// source.
func NewVerificationKeyDataSource() datasource.DataSource {
	return &VerificationKeyDataSource{}
}

// VerificationKeyDataSource reads the account ownership-verification record
// from the MXroute account.
type VerificationKeyDataSource struct {
	client *Client
}

// VerificationKeyDataSourceModel maps the mxroute_verification_key data source
// schema to Go values.
type VerificationKeyDataSourceModel struct {
	Key         types.String `tfsdk:"key"`
	Record      types.Object `tfsdk:"record"`
	Description types.String `tfsdk:"description"`
	ID          types.String `tfsdk:"id"`
}

// verificationRecordModel maps the {type, name, value} record to Go values.
type verificationRecordModel struct {
	Type  types.String `tfsdk:"type"`
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

// verificationRecordAttrTypes is the attribute type map for the record object.
func verificationRecordAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":  types.StringType,
		"name":  types.StringType,
		"value": types.StringType,
	}
}

func (d *VerificationKeyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_verification_key"
}

func (d *VerificationKeyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches the account ownership-verification record from the MXroute account — the verification key and the DNS record to publish to prove ownership.",
		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				MarkdownDescription: "The account verification key.",
				Computed:            true,
			},
			"record": schema.SingleNestedAttribute{
				MarkdownDescription: "The DNS record to publish to verify account ownership.",
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
			"description": schema.StringAttribute{
				MarkdownDescription: "A human-readable description of the verification record.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Data source identifier — the verification key.",
				Computed:            true,
			},
		},
	}
}

func (d *VerificationKeyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

func (d *VerificationKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config VerificationKeyDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// XXX: ENVELOPE-UNVERIFIED — the OpenAPI spec documents GET
	// /verification-key's 200 body as the VerificationKey object directly,
	// which may mean /verification-key is UNENVELOPED (no {success, data}
	// wrapper). client.Do always unwraps the "data" field, so if the endpoint
	// is in fact unenveloped, Do will read an empty/absent "data" and this
	// decode will silently yield a zero-valued VerificationKey. This code
	// assumes the STANDARD envelope for now (map via the shared
	// VerificationKey model). The enveloping MUST be verified against the live
	// account (the acceptance test below is the check): if the response comes
	// back unenveloped, the fix is a follow-up client change, not a change
	// here.
	var api VerificationKey

	if err := d.client.Do(ctx, http.MethodGet, "/verification-key", nil, &api); err != nil {
		resp.Diagnostics.AddError("Error reading account verification key", err.Error())

		return
	}

	record, diags := types.ObjectValueFrom(ctx, verificationRecordAttrTypes(), verificationRecordModel{
		Type:  types.StringValue(api.Record.Type),
		Name:  types.StringValue(api.Record.Name),
		Value: types.StringValue(api.Record.Value),
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := VerificationKeyDataSourceModel{
		Key:         types.StringValue(api.Key),
		Record:      record,
		Description: types.StringValue(api.Description),
		ID:          types.StringValue(api.Key),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
