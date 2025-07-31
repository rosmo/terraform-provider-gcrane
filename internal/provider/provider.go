// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package provider

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

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
	TempDir      types.String `tfsdk:"temporary_directory"`
}

type GcraneData struct {
	DockerConfig       string
	DockerConfigFile   string
	DockerIsConfigured atomic.Bool
	ConfigLock         sync.Mutex
	OriginalEnv        string
	Setup              func(ctx context.Context, data interface{}) error
	Cleanup            func(ctx context.Context, data interface{}) error
	Counter            atomic.Int32
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

This is a
[community maintained provider](https://www.terraform.io/docs/providers/type/community-index.html)
and not an official Google or Hashicorp product.
		`,
		Attributes: map[string]schema.Attribute{
			"docker_config": schema.StringAttribute{
				MarkdownDescription: "Contents of Docker config file (JSON)",
				Optional:            true,
			},
			"temporary_directory": schema.StringAttribute{
				MarkdownDescription: "Temporary directory for Docker config (uses system temp dir by default)",
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

	providerData := GcraneData{
		DockerConfigFile: "",
		DockerConfig:     data.DockerConfig.ValueString(),
		OriginalEnv:      os.Getenv("DOCKER_CONFIG"),
		Setup: func(ctx context.Context, data interface{}) error {
			gcraneData, ok := data.(GcraneData)
			if !ok {
				return fmt.Errorf("received unexpected data structure")
			}
			gcraneData.Counter.Add(1)
			if gcraneData.DockerConfig != "" && gcraneData.DockerConfigFile != "" && !gcraneData.DockerIsConfigured.Load() {
				gcraneData.DockerIsConfigured.Store(true)

				dockerConfigDir := filepath.Dir(gcraneData.DockerConfigFile)
				err := os.Mkdir(dockerConfigDir, 0700)
				if err != nil && !os.IsExist(err) {
					return fmt.Errorf("unable to create directory for Docker config %s: %s", dockerConfigDir, err.Error())
				}

				f, err := os.OpenFile(gcraneData.DockerConfigFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
				if err != nil {
					return fmt.Errorf("unable to create temporary file for Docker config %s: %s", gcraneData.DockerConfigFile, err.Error())
				}
				if _, err := f.Write([]byte(gcraneData.DockerConfig)); err != nil {
					return fmt.Errorf("unable to create temporary file for Docker config %s: %s", gcraneData.DockerConfigFile, err.Error())
				}
				if err := f.Close(); err != nil {
					return fmt.Errorf("unable to close temporary file for Docker config %s: %s", gcraneData.DockerConfigFile, err.Error())
				}

				gcraneData.ConfigLock.Lock()
				os.Setenv("DOCKER_CONFIG", dockerConfigDir)
				tflog.Trace(ctx, "Using temporary Docker config", map[string]interface{}{
					"directory": dockerConfigDir,
					"file":      gcraneData.DockerConfigFile,
				})
				gcraneData.ConfigLock.Unlock()
			}
			return nil
		},
		// Terrible emulation of provider teardown, see: https://github.com/hashicorp/terraform-plugin-sdk/issues/63
		Cleanup: func(ctx context.Context, data interface{}) error {
			gcraneData, ok := data.(GcraneData)
			if !ok {
				return fmt.Errorf("received unexpected data structure")
			}

			gcraneData.Counter.Add(-1)
			if gcraneData.Counter.Load() == 0 {
				if gcraneData.DockerConfig != "" && gcraneData.DockerConfigFile != "" && gcraneData.DockerIsConfigured.Load() {
					gcraneData.DockerIsConfigured.Store(false)

					gcraneData.ConfigLock.Lock()
					defer gcraneData.ConfigLock.Unlock()
					tflog.Trace(ctx, "Cleaning up temporary Docker config", map[string]interface{}{
						"file": gcraneData.DockerConfigFile,
					})
					err := os.Remove(gcraneData.DockerConfigFile)
					if err != nil {
						return fmt.Errorf("unable to delete temporary file for Docker config %s: %s", gcraneData.DockerConfigFile, err.Error())
					}
				}
				if gcraneData.OriginalEnv != "" {
					tflog.Trace(ctx, "Restoring original DOCKER_CONFIG", map[string]interface{}{
						"env": gcraneData.OriginalEnv,
					})

					os.Setenv("DOCKER_CONFIG", gcraneData.OriginalEnv)
				}
			}
			return nil
		},
	}

	if providerData.DockerConfig != "" {
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
		tempDir := os.TempDir()
		if data.TempDir.ValueString() != "" {
			tempDir = data.TempDir.ValueString()
		}
		tflog.Trace(ctx, "Temporary directory for Docker config", map[string]interface{}{
			"directory": tempDir,
		})
		dockerConfigDir := filepath.Join(tempDir, randomDir)
		dockerConfig := filepath.Join(dockerConfigDir, "config.json")
		providerData.DockerConfigFile = dockerConfig
	} else {
		tflog.Trace(ctx, "No docker.config specified")
	}

	resp.DataSourceData = &providerData
	resp.ResourceData = &providerData
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
