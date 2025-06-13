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
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/gcrane"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &CopyResource{}
var _ resource.ResourceWithImportState = &CopyResource{}

func NewCopyResource() resource.Resource {
	return &CopyResource{}
}

// CopyResource defines the resource implementation.
type CopyResource struct {
	Client *GcraneData
}

// CopyResourceModel describes the resource data model.
type CopyResourceModel struct {
	Recursive   types.Bool   `tfsdk:"recursive"`
	Source      types.String `tfsdk:"source"`
	Destination types.String `tfsdk:"destination"`
	Id          types.String `tfsdk:"id"`
}

func (r *CopyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_copy"
}

func (r *CopyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Copies container images between repositories",
		Description:         "Copies container images between repositories",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"recursive": schema.BoolAttribute{
				MarkdownDescription: "Recursive copy",
				Optional:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"source": schema.StringAttribute{
				MarkdownDescription: "Source for copy",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"destination": schema.StringAttribute{
				MarkdownDescription: "Destination for copy",
				Required:            true,
				//PlanModifiers: []planmodifier.String{
				//		stringplanmodifier.RequiresReplace(),
				//	},
			},
		},
	}
}

func (r *CopyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*GcraneData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *GcraneData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.Client = client
}

func (r *CopyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CopyResourceModel

	tflog.Trace(ctx, "Going to copy stuff", map[string]interface{}{
		"DOCKER_CONFIG": os.Getenv("DOCKER_CONFIG"),
	})

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var err error
	err = r.Client.Setup(ctx, *r.Client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not setup provider",
			err.Error(),
		)
		return
	}
	defer func() {
		err := r.Client.Cleanup(ctx, *r.Client)
		if err != nil {
			resp.Diagnostics.AddError(
				"Could not clean up provider",
				err.Error(),
			)
		}
	}()

	data.Id = data.Destination

	if data.Recursive.ValueBool() {
		err = gcrane.CopyRepository(ctx, data.Source.ValueString(), data.Destination.ValueString(), gcrane.WithContext(ctx))
	} else {
		err = gcrane.Copy(data.Source.ValueString(), data.Destination.ValueString(), gcrane.WithContext(ctx))
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not perform gcrane copy",
			fmt.Sprintf("Error when copying using gcrane: %s", err.Error()),
		)
		return
	}

	tflog.Trace(ctx, "Performed a copy using gcrane", map[string]interface{}{
		"recursive":   data.Recursive,
		"source":      data.Source,
		"destination": data.Destination,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CopyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CopyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CopyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CopyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CopyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CopyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *CopyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
