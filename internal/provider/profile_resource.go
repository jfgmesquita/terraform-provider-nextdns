package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jfgmesquita/terraform-provider-nextdns/internal/nextdns"
)

// Ensure ProfileResource satisfies the framework interfaces.
var _ resource.Resource = &ProfileResource{}
var _ resource.ResourceWithImportState = &ProfileResource{}

func NewProfileResource() resource.Resource {
	return &ProfileResource{}
}

// ProfileResource manages a NextDNS profile.
type ProfileResource struct {
	client *nextdns.Client
}

// ProfileResourceModel maps the resource schema to Go values.
type ProfileResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Fingerprint types.String `tfsdk:"fingerprint"`
}

func (r *ProfileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_profile"
}

func (r *ProfileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A NextDNS profile. Its settings, such as security and privacy, are managed by their own resources.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Profile ID, for example `abc123`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Profile name, as shown in the NextDNS dashboard.",
				Required:            true,
			},
			"fingerprint": schema.StringAttribute{
				MarkdownDescription: "Profile fingerprint, used in the DNS-over-HTTPS and DNS-over-TLS addresses.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ProfileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*nextdns.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *nextdns.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *ProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateProfile(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating NextDNS profile", err.Error())
		return
	}

	// The create response repeats the ID in "fingerprint", so read the
	// profile back to get the real fingerprint.
	profile, err := r.client.GetProfile(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading NextDNS profile after creating it", err.Error())
		return
	}

	data.ID = types.StringValue(profile.ID)
	data.Fingerprint = types.StringValue(profile.Fingerprint)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profile, err := r.client.GetProfile(ctx, data.ID.ValueString())
	if nextdns.IsNotFound(err) {
		// Deleted outside Terraform: forget it, so the next plan creates it again.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading NextDNS profile", err.Error())
		return
	}

	data.Name = types.StringValue(profile.Name)
	data.Fingerprint = types.StringValue(profile.Fingerprint)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.UpdateProfileName(ctx, data.ID.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating NextDNS profile", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteProfile(ctx, data.ID.ValueString())
	if err != nil && !nextdns.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting NextDNS profile", err.Error())
	}
}

func (r *ProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
