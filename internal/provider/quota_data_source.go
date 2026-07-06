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

// Ensure QuotaDataSource satisfies the framework interfaces.
var (
	_ datasource.DataSource              = &QuotaDataSource{}
	_ datasource.DataSourceWithConfigure = &QuotaDataSource{}
)

// NewQuotaDataSource returns a new mxroute_quota data source.
func NewQuotaDataSource() datasource.DataSource {
	return &QuotaDataSource{}
}

// QuotaDataSource reads the account-wide storage quota from the MXroute
// account.
type QuotaDataSource struct {
	client *Client
}

// QuotaDataSourceModel maps the mxroute_quota data source schema to Go values.
// grace_period is held as a framework object so a null grace period (the
// account is under quota) stays null rather than materializing an empty
// object.
type QuotaDataSourceModel struct {
	Username    types.String  `tfsdk:"username"`
	TotalUsed   types.Int64   `tfsdk:"total_used"`
	TotalLimit  types.Int64   `tfsdk:"total_limit"`
	PercentUsed types.Float64 `tfsdk:"percent_used"`
	Breakdown   types.Object  `tfsdk:"breakdown"`
	GracePeriod types.Object  `tfsdk:"grace_period"`
	UpdatedAt   types.String  `tfsdk:"updated_at"`
	ID          types.String  `tfsdk:"id"`
}

// quotaBreakdownModel maps the breakdown object to Go values.
type quotaBreakdownModel struct {
	Email     types.Int64 `tfsdk:"email"`
	Web       types.Int64 `tfsdk:"web"`
	Databases types.Int64 `tfsdk:"databases"`
	Backups   types.Int64 `tfsdk:"backups"`
	Other     types.Int64 `tfsdk:"other"`
}

// quotaGracePeriodModel maps the grace_period object to Go values.
type quotaGracePeriodModel struct {
	DaysRemaining types.Int64  `tfsdk:"days_remaining"`
	Deadline      types.String `tfsdk:"deadline"`
}

// quotaBreakdownAttrTypes is the attribute type map for the breakdown object.
func quotaBreakdownAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"email":     types.Int64Type,
		"web":       types.Int64Type,
		"databases": types.Int64Type,
		"backups":   types.Int64Type,
		"other":     types.Int64Type,
	}
}

// quotaGracePeriodAttrTypes is the attribute type map for the grace_period
// object.
func quotaGracePeriodAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"days_remaining": types.Int64Type,
		"deadline":       types.StringType,
	}
}

func (d *QuotaDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_quota"
}

func (d *QuotaDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches the account-wide storage quota from the MXroute account, including per-category usage and any active over-quota grace period.",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				MarkdownDescription: "The account username the quota belongs to.",
				Computed:            true,
			},
			"total_used": schema.Int64Attribute{
				MarkdownDescription: "Total storage used across the account, in bytes.",
				Computed:            true,
			},
			"total_limit": schema.Int64Attribute{
				MarkdownDescription: "Total storage limit for the account, in bytes (0 means unlimited).",
				Computed:            true,
			},
			"percent_used": schema.Float64Attribute{
				MarkdownDescription: "Fraction of the total limit currently used, as a percentage.",
				Computed:            true,
			},
			"breakdown": schema.SingleNestedAttribute{
				MarkdownDescription: "Per-category storage usage, in bytes.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"email": schema.Int64Attribute{
						MarkdownDescription: "Storage used by email, in bytes.",
						Computed:            true,
					},
					"web": schema.Int64Attribute{
						MarkdownDescription: "Storage used by web hosting, in bytes.",
						Computed:            true,
					},
					"databases": schema.Int64Attribute{
						MarkdownDescription: "Storage used by databases, in bytes.",
						Computed:            true,
					},
					"backups": schema.Int64Attribute{
						MarkdownDescription: "Storage used by backups, in bytes.",
						Computed:            true,
					},
					"other": schema.Int64Attribute{
						MarkdownDescription: "Storage used by other categories, in bytes.",
						Computed:            true,
					},
				},
			},
			"grace_period": schema.SingleNestedAttribute{
				MarkdownDescription: "The over-quota grace period, or null when the account is within its limit.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"days_remaining": schema.Int64Attribute{
						MarkdownDescription: "Days remaining in the grace period before enforcement.",
						Computed:            true,
					},
					"deadline": schema.StringAttribute{
						MarkdownDescription: "The grace period deadline.",
						Computed:            true,
					},
				},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "When the quota figures were last computed.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Data source identifier — the account username.",
				Computed:            true,
			},
		},
	}
}

func (d *QuotaDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

func (d *QuotaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config QuotaDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// XXX: ENVELOPE-UNVERIFIED — the OpenAPI spec documents GET /quota's 200
	// body as the Quota object directly, which may mean /quota is UNENVELOPED
	// (no {success, data} wrapper). client.Do always unwraps the "data" field,
	// so if /quota is in fact unenveloped, Do will read an empty/absent "data"
	// and this decode will silently yield a zero-valued Quota. This code
	// assumes the STANDARD envelope for now (map via the shared Quota model).
	// The enveloping MUST be verified against the live account (the acceptance
	// test below is the check): if the response comes back unenveloped, the
	// fix is a follow-up client change, not a change here.
	var api Quota

	if err := d.client.Do(ctx, http.MethodGet, "/quota", nil, &api); err != nil {
		resp.Diagnostics.AddError("Error reading account quota", err.Error())

		return
	}

	breakdown, diags := types.ObjectValueFrom(ctx, quotaBreakdownAttrTypes(), quotaBreakdownModel{
		Email:     types.Int64Value(api.Breakdown.Email),
		Web:       types.Int64Value(api.Breakdown.Web),
		Databases: types.Int64Value(api.Breakdown.Databases),
		Backups:   types.Int64Value(api.Breakdown.Backups),
		Other:     types.Int64Value(api.Breakdown.Other),
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	gracePeriod, diags := quotaGracePeriodObject(ctx, api.GracePeriod)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := QuotaDataSourceModel{
		Username:    types.StringValue(api.Username),
		TotalUsed:   types.Int64Value(api.TotalUsed),
		TotalLimit:  types.Int64Value(api.TotalLimit),
		PercentUsed: types.Float64Value(api.PercentUsed),
		Breakdown:   breakdown,
		GracePeriod: gracePeriod,
		UpdatedAt:   types.StringValue(api.UpdatedAt),
		ID:          types.StringValue(api.Username),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// quotaGracePeriodObject builds a framework object for the grace period,
// returning a null object when the API returned null (the account is within
// its limit).
func quotaGracePeriodObject(ctx context.Context, api *QuotaGracePeriod) (types.Object, diag.Diagnostics) {
	if api == nil {
		return types.ObjectNull(quotaGracePeriodAttrTypes()), nil
	}

	return types.ObjectValueFrom(ctx, quotaGracePeriodAttrTypes(), quotaGracePeriodModel{
		DaysRemaining: types.Int64Value(api.DaysRemaining),
		Deadline:      types.StringValue(api.Deadline),
	})
}
