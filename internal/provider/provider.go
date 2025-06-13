// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure GcraneProvider satisfies various provider interfaces.
var _ provider.Provider = &GcraneProvider{}
var _ provider.ProviderWithFunctions = &GcraneProvider{}
var _ provider.ProviderWithEphemeralResources = &GcraneProvider{}

// GcraneProvider defines the provider implementation.
type GcraneProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// GcraneProviderModel describes the provider data model.
type GcraneProviderModel struct {
	DockerConfig types.String `tfsdk:"docker_config"`
}

func (p *GcraneProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "gcrane"
	resp.Version = p.version
}

func (p *GcraneProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for gcrane.",
		Attributes: map[string]schema.Attribute{
			"docker_config": schema.StringAttribute{
				MarkdownDescription: "Contents of docker.config",
				Optional:            true,
			},
		},
	}
}

func (p *GcraneProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data GcraneProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.DockerConfig.ValueString() != "" {
		f, err := os.CreateTemp("", "docker.config")
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating temporary docker.config",
				"Unable to create temporary file for docker.config",
			)
		}
		defer os.Remove(f.Name()) // clean up

		if _, err := f.Write([]byte(data.DockerConfig.ValueString())); err != nil {
			resp.Diagnostics.AddError(
				"Unable to write temporary docker.config",
				fmt.Sprintf("Unable to create temporary file for docker.config: %s", f.Name()),
			)
		}
		if err := f.Close(); err != nil {
			resp.Diagnostics.AddError(
				"Unable to close temporary docker.config",
				fmt.Sprintf("Unable to close temporary file for docker.config: %s", f.Name()),
			)
		}
		defer os.Remove(f.Name())

		os.Setenv("DOCKER_CONFIG", f.Name())
	}

	client := http.DefaultClient
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *GcraneProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCopyResource,
	}
}

func (p *GcraneProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *GcraneProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewGcraneListDataSource,
	}
}

func (p *GcraneProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &GcraneProvider{
			version: version,
		}
	}
}
