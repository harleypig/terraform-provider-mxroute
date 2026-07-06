package provider

import (
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// computedIDAttribute returns the standard resource `id` attribute: a
// computed string held stable across plans with UseStateForUnknown. desc is
// the per-resource MarkdownDescription.
func computedIDAttribute(desc string) rschema.StringAttribute {
	return rschema.StringAttribute{
		MarkdownDescription: desc,
		Computed:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

// dataSourceIDAttribute returns the standard data-source `id` attribute: a
// plain computed string (no plan modifier — data sources are read each plan).
func dataSourceIDAttribute(desc string) dschema.StringAttribute {
	return dschema.StringAttribute{
		MarkdownDescription: desc,
		Computed:            true,
	}
}

// requiredReplaceString returns a required string attribute that forces
// replacement when changed — the shape used for the immutable identity
// inputs (domain, alias, entry, …) the MXroute API cannot update in place.
func requiredReplaceString(desc string) rschema.StringAttribute {
	return rschema.StringAttribute{
		MarkdownDescription: desc,
		Required:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
}
