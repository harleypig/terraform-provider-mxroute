package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure PointerResource satisfies the framework interfaces.
var (
	_ resource.Resource                = &PointerResource{}
	_ resource.ResourceWithConfigure   = &PointerResource{}
	_ resource.ResourceWithImportState = &PointerResource{}
)

// NewPointerResource returns a new mxroute_pointer resource.
func NewPointerResource() resource.Resource {
	return &PointerResource{}
}

// PointerResource manages a domain pointer on the MXroute account.
type PointerResource struct {
	client *Client
}

// PointerResourceModel maps the mxroute_pointer schema to Go values.
type PointerResourceModel struct {
	Domain  types.String `tfsdk:"domain"`
	Pointer types.String `tfsdk:"pointer"`
	Alias   types.Bool   `tfsdk:"alias"`
	Type    types.String `tfsdk:"type"`
	Target  types.String `tfsdk:"target"`
	ID      types.String `tfsdk:"id"`
}

// pointerAPIModel is the MXroute API representation of a domain pointer, as
// returned by the list endpoint.
type pointerAPIModel struct {
	Pointer string `json:"pointer"`
	Type    string `json:"type"`
	Target  string `json:"target"`
}

// createPointerRequest is the POST /domains/{domain}/pointers body.
type createPointerRequest struct {
	Pointer string `json:"pointer"`
	Alias   bool   `json:"alias"`
}

func (r *PointerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pointer"
}

func (r *PointerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a pointer (alias or redirect) for a mail domain on the MXroute account. MXroute exposes no in-place update for a pointer, so changing any attribute replaces the resource.",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "The parent domain the pointer belongs to (e.g. `example.com`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"pointer": schema.StringAttribute{
				MarkdownDescription: "The pointer name that resolves to the parent domain (e.g. `www.example.com`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"alias": schema.BoolAttribute{
				MarkdownDescription: "Whether the pointer is an alias (`true`) or a redirect (`false`). Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The pointer type reported by the API — `alias` or `redirect`.",
				Computed:            true,
			},
			"target": schema.StringAttribute{
				MarkdownDescription: "The target the pointer resolves to.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier — `<domain>/<pointer>`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *PointerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PointerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PointerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := plan.Domain.ValueString()
	pointer := plan.Pointer.ValueString()

	body := createPointerRequest{
		Pointer: pointer,
		Alias:   plan.Alias.ValueBool(),
	}

	if err := r.client.Do(ctx, http.MethodPost, "/domains/"+domain+"/pointers", body, nil); err != nil {
		resp.Diagnostics.AddError("Error creating pointer", err.Error())

		return
	}

	// The create response carries no data; read the pointer back from the
	// list to populate the computed attributes.
	api, err := r.fetchPointer(ctx, domain, pointer)
	if err != nil {
		resp.Diagnostics.AddError("Error reading pointer after create", err.Error())

		return
	}

	if api == nil {
		resp.Diagnostics.AddError("Error reading pointer after create", fmt.Sprintf("pointer %q was not found on domain %q immediately after creation", pointer, domain))

		return
	}

	state := pointerModelFromAPI(domain, api)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PointerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PointerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := state.Domain.ValueString()
	pointer := state.Pointer.ValueString()

	api, err := r.fetchPointer(ctx, domain, pointer)
	if err != nil {
		resp.Diagnostics.AddError("Error reading pointer", err.Error())

		return
	}

	if api == nil {
		resp.State.RemoveResource(ctx)

		return
	}

	newState := pointerModelFromAPI(domain, api)

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Update refreshes the computed attributes. Every configurable attribute is
// RequiresReplace, so a plan never reaches Update with a changed value — it
// re-reads to keep state accurate.
func (r *PointerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PointerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := plan.Domain.ValueString()
	pointer := plan.Pointer.ValueString()

	api, err := r.fetchPointer(ctx, domain, pointer)
	if err != nil {
		resp.Diagnostics.AddError("Error reading pointer", err.Error())

		return
	}

	if api == nil {
		resp.Diagnostics.AddError("Error reading pointer", fmt.Sprintf("pointer %q was not found on domain %q", pointer, domain))

		return
	}

	state := pointerModelFromAPI(domain, api)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PointerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PointerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := state.Domain.ValueString()
	pointer := state.Pointer.ValueString()

	// A pointer already gone is a successful delete.
	if err := r.client.Do(ctx, http.MethodDelete, "/domains/"+domain+"/pointers/"+pointer, nil, nil); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting pointer", err.Error())

		return
	}
}

func (r *PointerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domain, pointer, found := strings.Cut(req.ID, "/")
	if !found || domain == "" || pointer == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier of the form \"domain/pointer\", got: %q", req.ID),
		)

		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), domain)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("pointer"), pointer)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// fetchPointer lists the domain's pointers and returns the one named pointer,
// or (nil, nil) when the domain or the pointer does not exist.
func (r *PointerResource) fetchPointer(ctx context.Context, domain, pointer string) (*pointerAPIModel, error) {
	var list []pointerAPIModel

	if err := r.client.Do(ctx, http.MethodGet, "/domains/"+domain+"/pointers", nil, &list); err != nil {
		if IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	for i := range list {
		if list[i].Pointer == pointer {
			return &list[i], nil
		}
	}

	return nil, nil
}

// pointerModelFromAPI maps an API pointer onto the Terraform state model. The
// alias flag is derived from the reported type: "alias" is an alias, anything
// else (e.g. "redirect") is not.
func pointerModelFromAPI(domain string, api *pointerAPIModel) PointerResourceModel {
	return PointerResourceModel{
		Domain:  types.StringValue(domain),
		Pointer: types.StringValue(api.Pointer),
		Alias:   types.BoolValue(api.Type == "alias"),
		Type:    types.StringValue(api.Type),
		Target:  types.StringValue(api.Target),
		ID:      types.StringValue(domain + "/" + api.Pointer),
	}
}
