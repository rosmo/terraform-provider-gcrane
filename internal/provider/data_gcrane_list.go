// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/google/go-containerregistry/pkg/gcrane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &GcraneListDataSource{}

func NewGcraneListDataSource() datasource.DataSource {
	return &GcraneListDataSource{}
}

// GcraneListDataSource defines the data source implementation.
type GcraneListDataSource struct {
	Client *GcraneData
}

type GcraneListDataSourceImageModel struct {
	ImageSizeBytes types.Int64  `tfsdk:"image_size_bytes"`
	MediaType      types.String `tfsdk:"media_type"`
	Created        types.Int64  `tfsdk:"time_created_ms"`
	Uploaded       types.Int64  `tfsdk:"time_uploaded_ms"`
	Tags           types.Set    `tfsdk:"tags"`
}

type GcraneListDataSourceImagesModel struct {
	Manifests types.Map `tfsdk:"manifests"`
	Tags      types.Set `tfsdk:"tags"`
	Children  types.Set `tfsdk:"children"`
}

// GcraneListDataSourceModel describes the data source data model.
type GcraneListDataSourceModel struct {
	Repository types.String   `tfsdk:"repository"`
	Id         types.String   `tfsdk:"id"`
	Images     []types.Object `tfsdk:"images"`
}

func (o GcraneListDataSourceImageModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"image_size_bytes": types.Int64Type,
		"media_type":       types.StringType,
		"time_created_ms":  types.Int64Type,
		"time_uploaded_ms": types.Int64Type,
		"tags": types.SetType{
			ElemType: types.StringType,
		},
	}
}

func (o GcraneListDataSourceImagesModel) AttributeTypes() map[string]attr.Type {
	imageModel := GcraneListDataSourceImageModel{}
	return map[string]attr.Type{
		"manifests": types.MapType{
			ElemType: types.ObjectType{
				AttrTypes: imageModel.AttributeTypes(),
			},
		},
		"tags": types.SetType{
			ElemType: types.StringType,
		},
		"children": types.SetType{
			ElemType: types.StringType,
		},
	}
}

func (d *GcraneListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_list"
}

func (d *GcraneListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Fetch a list of container images from repository",
		MarkdownDescription: "Fetch a list of container images from repository",

		Attributes: map[string]schema.Attribute{
			"repository": schema.StringAttribute{
				MarkdownDescription: "Repository address",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier",
				Computed:            true,
			},
			"images": schema.SetNestedAttribute{
				MarkdownDescription: "Output of list operation",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"manifests": schema.MapNestedAttribute{
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"image_size_bytes": schema.Int64Attribute{
										Computed: true,
									},
									"media_type": schema.StringAttribute{
										Computed: true,
									},
									"time_created_ms": schema.Int64Attribute{
										Computed: true,
									},
									"time_uploaded_ms": schema.Int64Attribute{
										Computed: true,
									},
									"tags": schema.SetAttribute{
										ElementType: types.StringType,
										Computed:    true,
									},
								},
							},
							Computed: true,
						},
						"children": schema.SetAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
						"tags": schema.SetAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *GcraneListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*GcraneData)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *GcraneData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.Client = client
}

func (d *GcraneListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GcraneListDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var err error
	err = d.Client.Setup(ctx, *d.Client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not setup provider",
			err.Error(),
		)
		return
	}
	defer func() {
		err := d.Client.Cleanup(ctx, *d.Client)
		if err != nil {
			resp.Diagnostics.AddError(
				"Could not clean up provider",
				err.Error(),
			)
		}
	}()

	data.Id = data.Repository

	repo, err := name.NewRepository(data.Repository.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read repository",
			fmt.Sprintf("Failed to read repository %s: %s", data.Repository.ValueString(), err.Error()),
		)
		return
	}

	opts := []google.Option{
		google.WithAuthFromKeychain(gcrane.Keychain),
		google.WithContext(ctx),
	}

	tags, err := google.List(repo, opts...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to list repository",
			fmt.Sprintf("Failed to list repository %s: %s", data.Repository.ValueString(), err.Error()),
		)
		return
	}

	childList, diags := types.SetValueFrom(ctx, types.StringType, tags.Children)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	topTagsList, diags := types.SetValueFrom(ctx, types.StringType, tags.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	images := GcraneListDataSourceImagesModel{
		Children: childList,
		Tags:     topTagsList,
	}

	manifestsMap := make(map[string]GcraneListDataSourceImageModel, 0)
	for k, v := range tags.Manifests {
		tagsList, diags := types.SetValueFrom(ctx, types.StringType, v.Tags)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		manifest := GcraneListDataSourceImageModel{
			ImageSizeBytes: types.Int64Value(int64(v.Size)),
			MediaType:      types.StringValue(v.MediaType),
			Created:        types.Int64Value(v.Created.UnixMilli()),
			Uploaded:       types.Int64Value(v.Uploaded.UnixMilli()),
			Tags:           tagsList,
		}
		manifestsMap[k] = manifest
	}
	manifestMapValue, diags := types.MapValueFrom(ctx, types.ObjectType{AttrTypes: GcraneListDataSourceImageModel{}.AttributeTypes()}, manifestsMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	images.Manifests = manifestMapValue

	imagesObject, diags := types.ObjectValueFrom(ctx, images.AttributeTypes(), images)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Images = append(data.Images, imagesObject)

	if len(tags.Manifests) == 0 && len(tags.Children) == 0 {
		for _, tag := range tags.Tags {
			tflog.Trace(ctx, fmt.Sprintf("FOO %s:%s\n", repo, tag))
		}
	} else {
		tflog.Trace(ctx, fmt.Sprintf("FOO manifests %v, children: %v: tags: %v\n", tags.Manifests, tags.Children, tags.Tags))
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "read a data source", map[string]interface{}{
		"repository": data.Repository,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
