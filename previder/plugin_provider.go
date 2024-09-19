package previder

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/previder/terraform-provider-previder/internal/kubernetes_cluster"
	"github.com/previder/terraform-provider-previder/internal/staas_environment"
	"github.com/previder/terraform-provider-previder/internal/virtual_firewall"
	"github.com/previder/terraform-provider-previder/internal/virtual_network"
	"github.com/previder/terraform-provider-previder/internal/virtual_server"
)

type PreviderProvider struct{}

var _ provider.Provider = &PreviderProvider{}
var version = provider.MetadataResponse{Version: "not built yet"}

func NewPreviderProvider() provider.Provider {
	return &PreviderProvider{}
}

// Metadata should return the metadata for the provider, such as
// a type name and version data.
//
// Implementing the MetadataResponse.TypeName will populate the
// datasource.MetadataRequest.ProviderTypeName and
// resource.MetadataRequest.ProviderTypeName fields automatically.
func (p *PreviderProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "previder"
	resp.Version = version.Version
}

// Schema should return the schema for this provider.
func (p *PreviderProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The token key for API operations.",
			},
			"url": schema.StringAttribute{
				Optional:    true,
				Description: "The API endpoint URL.",
			},
			"customer": schema.StringAttribute{
				Optional:    true,
				Description: "An optional sub customer object id",
			},
		},
	}
}

func (p *PreviderProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {

	var data Config

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config := Config{
		Token:      data.Token,
		Url:        data.Url,
		CustomerId: data.CustomerId,
	}

	var baseClient = config.Client()
	resp.DataSourceData = baseClient
	resp.ResourceData = baseClient

	tflog.Info(ctx, "Previder Client configured", map[string]any{"url": config.Url.ValueString(), "customer": config.CustomerId.ValueString()})
	tflog.Info(ctx, "terraform-provider-previder info", map[string]any{"version": version.Version})
}

// DataSources returns a slice of functions to instantiate each DataSource
// implementation.
//
// The data source type name is determined by the DataSource implementing
// the Metadata method. All data sources must have unique names.
func (p *PreviderProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

// Resources returns a slice of functions to instantiate each Resource
// implementation.
//
// The resource type name is determined by the Resource implementing
// the Metadata method. All resources must have unique names.
func (p *PreviderProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		virtual_server.NewResource,
		virtual_network.NewResource,
		virtual_firewall.NewResource,
		kubernetes_cluster.NewResource,
		staas_environment.NewResource,
	}
}
