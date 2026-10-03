// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

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

	"github.com/jfgmesquita/terraform-provider-nextdns/internal/nextdns"
)

// Ensure NextDNSProvider satisfies the provider interface.
var _ provider.Provider = &NextDNSProvider{}

// NextDNSProvider defines the provider implementation.
type NextDNSProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// NextDNSProviderModel describes the provider data model.
type NextDNSProviderModel struct {
	APIKey types.String `tfsdk:"api_key"`
}

func (p *NextDNSProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "nextdns"
	resp.Version = p.version
}

func (p *NextDNSProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage NextDNS profiles.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "NextDNS API key, from https://my.nextdns.io/account. Can also be set with the `NEXTDNS_API_KEY` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *NextDNSProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data NextDNSProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown NextDNS API key",
			"The api_key value is not known yet. Set it to a static value or use the NEXTDNS_API_KEY environment variable.",
		)
		return
	}

	apiKey := os.Getenv("NEXTDNS_API_KEY")
	if !data.APIKey.IsNull() {
		apiKey = data.APIKey.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing NextDNS API key",
			"Set the api_key provider attribute or the NEXTDNS_API_KEY environment variable. "+
				"Create an API key at https://my.nextdns.io/account.",
		)
		return
	}

	// Resources and data sources receive this client in their own Configure.
	client := nextdns.NewClient(apiKey)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *NextDNSProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProfileResource,
		NewSecurityResource,
		NewPrivacyResource,
	}
}

func (p *NextDNSProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &NextDNSProvider{
			version: version,
		}
	}
}
