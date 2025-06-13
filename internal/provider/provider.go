// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"crypto/rand"
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
		MarkdownDescription: `Terraform provider for [gcrane](https://github.com/google/go-containerregistry/blob/main/cmd/gcrane/README.md).

Allows copying images between Docker registries and also fetching some details (like images, tags, etc).
Does not require gcrane or Docker installed. You can specify a Docker config JSON file as a string
in the provider configuration block, which will then be used during operations.
		`,
		Attributes: map[string]schema.Attribute{
			"docker_config": schema.StringAttribute{
				MarkdownDescription: "Contents of Docker config file (JSON)",
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
		randBytes := make([]byte, 16)
		_, err := rand.Read(randBytes)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating randomness for temporary Docker config",
				fmt.Sprintf("Unable to randomness Docker config: %s", err.Error()),
			)
			return
		}
		randomDir := hex.EncodeToString(randBytes)
		dockerConfigDir := filepath.Join(os.TempDir(), randomDir)

		err = os.Mkdir(dockerConfigDir, 0700)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating directory for temporary Docker config",
				fmt.Sprintf("Unable to create directory for Docker config %s: %s", dockerConfigDir, err.Error()),
			)
			return
		}

		dockerConfig := filepath.Join(dockerConfigDir, "config.json")
		f, err := os.OpenFile(dockerConfig, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating temporary docker.config",
				fmt.Sprintf("Unable to create temporary file for Docker config %s: %s", dockerConfig, err.Error()),
			)
			return
		}
		if _, err := f.Write([]byte(data.DockerConfig.ValueString())); err != nil {
			resp.Diagnostics.AddError(
				"Unable to write temporary Docker config",
				fmt.Sprintf("Unable to create temporary file for Docker config %s: %s", dockerConfig, err.Error()),
			)
			return
		}
		if err := f.Close(); err != nil {
			resp.Diagnostics.AddError(
				"Unable to close temporary Docker config",
				fmt.Sprintf("Unable to close temporary file for Docker config %s: %s", dockerConfig, err.Error()),
			)
			return
		}

		tflog.Trace(ctx, "Temporary Docker config created", map[string]interface{}{
			"filename": dockerConfig,
		})
		os.Setenv("DOCKER_CONFIG", dockerConfigDir)
	} else {
		tflog.Trace(ctx, "No docker.config specified")
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
