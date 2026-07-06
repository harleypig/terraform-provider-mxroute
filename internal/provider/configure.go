package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// configureResourceClient extracts the *Client a resource needs from its
// ConfigureRequest. It returns nil — after adding a diagnostic when the
// data is present but the wrong type — if the provider has not supplied
// client data yet, in which case the caller keeps its zero client and the
// framework calls Configure again once the data is available.
func configureResourceClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *Client {
	if req.ProviderData == nil {
		return nil
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return nil
	}

	return client
}

// configureDataSourceClient is the data-source counterpart of
// configureResourceClient.
func configureDataSourceClient(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) *Client {
	if req.ProviderData == nil {
		return nil
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return nil
	}

	return client
}
