package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure DomainResource satisfies the framework interfaces.
var (
	_ resource.Resource                = &DomainResource{}
	_ resource.ResourceWithConfigure   = &DomainResource{}
	_ resource.ResourceWithImportState = &DomainResource{}
)

// NewDomainResource returns a new mxroute_domain resource.
func NewDomainResource() resource.Resource {
	return &DomainResource{}
}

// DomainResource manages a mail domain on the MXroute account.
type DomainResource struct {
	client *Client
}

// DomainResourceModel maps the mxroute_domain schema to Go values.
type DomainResourceModel struct {
	Domain      types.String `tfsdk:"domain"`
	MailHosting types.Bool   `tfsdk:"mail_hosting"`
	SSLEnabled  types.Bool   `tfsdk:"ssl_enabled"`
	Pointers    types.List   `tfsdk:"pointers"`
	ID          types.String `tfsdk:"id"`
}

// domainAPIModel is the MXroute API representation of a domain.
type domainAPIModel struct {
	Domain      string   `json:"domain"`
	MailHosting bool     `json:"mail_hosting"`
	SSLEnabled  bool     `json:"ssl_enabled"`
	Pointers    []string `json:"pointers"`
}

// createDomainRequest is the POST /domains body.
type createDomainRequest struct {
	Domain string `json:"domain"`
}

func (r *DomainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *DomainResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a mail domain on the MXroute account. MXroute exposes no in-place update for a domain, so changing `domain` replaces the resource.",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain name to host mail for (e.g. `example.com`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mail_hosting": schema.BoolAttribute{
				MarkdownDescription: "Whether mail hosting is enabled for the domain.",
				Computed:            true,
			},
			"ssl_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether SSL is enabled for the domain.",
				Computed:            true,
			},
			"pointers": schema.ListAttribute{
				MarkdownDescription: "Domain pointers (aliases) that resolve to this domain.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier — the domain name.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *DomainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *DomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := plan.Domain.ValueString()

	if err := r.client.Do(ctx, http.MethodPost, "/domains", createDomainRequest{Domain: domain}, nil); err != nil {
		resp.Diagnostics.AddError("Error creating domain", err.Error())

		return
	}

	// The create response is partial; read the domain back to populate the
	// computed attributes.
	api, err := r.fetchDomain(ctx, domain)
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain after create", err.Error())

		return
	}

	if api == nil {
		resp.Diagnostics.AddError("Error reading domain after create", fmt.Sprintf("domain %q was not found immediately after creation", domain))

		return
	}

	state, diags := domainModelFromAPI(ctx, api)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	api, err := r.fetchDomain(ctx, state.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain", err.Error())

		return
	}

	if api == nil {
		resp.State.RemoveResource(ctx)

		return
	}

	newState, diags := domainModelFromAPI(ctx, api)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Update refreshes the computed attributes. The domain attribute is
// RequiresReplace and every other attribute is computed, so a plan never
// reaches Update with a changed value — it re-reads to keep state accurate.
func (r *DomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	api, err := r.fetchDomain(ctx, plan.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain", err.Error())

		return
	}

	if api == nil {
		resp.Diagnostics.AddError("Error reading domain", fmt.Sprintf("domain %q was not found", plan.Domain.ValueString()))

		return
	}

	state, diags := domainModelFromAPI(ctx, api)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A domain already gone is a successful delete.
	if err := r.client.Do(ctx, http.MethodDelete, "/domains/"+state.Domain.ValueString(), nil, nil); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting domain", err.Error())

		return
	}
}

func (r *DomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// fetchDomain GETs a domain, returning (nil, nil) when it does not exist.
func (r *DomainResource) fetchDomain(ctx context.Context, domain string) (*domainAPIModel, error) {
	var api domainAPIModel

	if err := r.client.Do(ctx, http.MethodGet, "/domains/"+domain, nil, &api); err != nil {
		if IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return &api, nil
}

// domainModelFromAPI maps an API domain onto the Terraform state model.
func domainModelFromAPI(ctx context.Context, api *domainAPIModel) (DomainResourceModel, diag.Diagnostics) {
	pointers, diags := types.ListValueFrom(ctx, types.StringType, api.Pointers)

	return DomainResourceModel{
		Domain:      types.StringValue(api.Domain),
		MailHosting: types.BoolValue(api.MailHosting),
		SSLEnabled:  types.BoolValue(api.SSLEnabled),
		Pointers:    pointers,
		ID:          types.StringValue(api.Domain),
	}, diags
}
