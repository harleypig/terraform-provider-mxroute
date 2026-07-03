package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure MxrouteProvider satisfies the provider interface.
var _ provider.Provider = &MxrouteProvider{}

// MxrouteProvider defines the provider implementation.
type MxrouteProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// MxrouteProviderModel describes the provider data model.
type MxrouteProviderModel struct {
	Server   types.String `tfsdk:"server"`
	Username types.String `tfsdk:"username"`
	APIKey   types.String `tfsdk:"api_key"`
}

func (p *MxrouteProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "mxroute"
	resp.Version = p.version
}

func (p *MxrouteProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"server": schema.StringAttribute{
				MarkdownDescription: "MXroute mail server hostname, sent as `X-Server`. Falls back to the `MXROUTE_SERVER` environment variable.",
				Optional:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "DirectAdmin username, sent as `X-Username`. Falls back to the `MXROUTE_USERNAME` environment variable.",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API key, sent as `X-API-Key`. Falls back to the `MXROUTE_API_KEY` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *MxrouteProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data MxrouteProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// A configured attribute wins over the environment fallback.
	server := os.Getenv("MXROUTE_SERVER")
	if !data.Server.IsNull() {
		server = data.Server.ValueString()
	}

	username := os.Getenv("MXROUTE_USERNAME")
	if !data.Username.IsNull() {
		username = data.Username.ValueString()
	}

	apiKey := os.Getenv("MXROUTE_API_KEY")
	if !data.APIKey.IsNull() {
		apiKey = data.APIKey.ValueString()
	}

	if server == "" {
		resp.Diagnostics.AddAttributeError(path.Root("server"), "Missing MXroute server",
			"Set the `server` attribute or the MXROUTE_SERVER environment variable.")
	}

	if username == "" {
		resp.Diagnostics.AddAttributeError(path.Root("username"), "Missing MXroute username",
			"Set the `username` attribute or the MXROUTE_USERNAME environment variable.")
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(path.Root("api_key"), "Missing MXroute API key",
			"Set the `api_key` attribute or the MXROUTE_API_KEY environment variable.")
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client := NewClient(ClientConfig{
		Server:   server,
		Username: username,
		APIKey:   apiKey,
	})

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *MxrouteProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDomainResource,
		NewEmailAccountResource,
		NewForwarderResource,
		NewPointerResource,
		NewCatchAllResource,
		NewSpamSettingsResource,
		NewSpamBlacklistEntryResource,
		NewSpamWhitelistEntryResource,
		NewResellerPackageResource,
		NewResellerUserResource,
	}
}

func (p *MxrouteProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDomainDataSource,
		NewDNSDataSource,
		NewQuotaDataSource,
		NewEmailQuotaDataSource,
		NewVerificationKeyDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &MxrouteProvider{
			version: version,
		}
	}
}
