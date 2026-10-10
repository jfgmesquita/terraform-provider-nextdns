package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jfgmesquita/terraform-provider-nextdns/internal/nextdns"
)

// Ensure PrivacyResource satisfies the framework interfaces.
var _ resource.Resource = &PrivacyResource{}
var _ resource.ResourceWithImportState = &PrivacyResource{}

func NewPrivacyResource() resource.Resource {
	return &PrivacyResource{}
}

// PrivacyResource manages the privacy settings of a NextDNS profile.
type PrivacyResource struct {
	client *nextdns.Client
}

// PrivacyResourceModel maps the resource schema to Go values.
type PrivacyResourceModel struct {
	ProfileID         types.String `tfsdk:"profile_id"`
	Blocklists        types.Set    `tfsdk:"blocklists"`
	Natives           types.Set    `tfsdk:"natives"`
	DisguisedTrackers types.Bool   `tfsdk:"disguised_trackers"`
	AllowAffiliate    types.Bool   `tfsdk:"allow_affiliate"`
}

func (r *PrivacyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privacy"
}

func (r *PrivacyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The privacy settings of a NextDNS profile.\n\n" +
			"~> Removing this resource from the configuration does not change the privacy settings in NextDNS: " +
			"Terraform only stops managing them. To turn settings off, set them to `false` or remove them from the resource.",

		Attributes: map[string]schema.Attribute{
			"profile_id": schema.StringAttribute{
				MarkdownDescription: "ID of the profile these settings belong to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"blocklists": schema.SetAttribute{
				MarkdownDescription: "Block ads & trackers using the most popular blocklists available — all updated in real time. " +
					"Values are blocklist IDs, for example, `[\"nextdns-recommended\", \"oisd\"]`. " +
					"The available IDs are listed at https://api.nextdns.io/privacy/blocklists.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(emptyStringSet()),
			},
			"natives": schema.SetAttribute{
				MarkdownDescription: "Block trackers — often operating at the operating system level — " +
					"that monitor a broad range of your activity on a device. This could include all the websites you visit, " +
					"everything you type or your location at all times. " +
					"Values are vendor IDs, for example, `[\"apple\", \"windows\"]`. " +
					"The available IDs are listed at https://api.nextdns.io/privacy/natives.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(emptyStringSet()),
			},
			"disguised_trackers": schema.BoolAttribute{
				MarkdownDescription: "Automatically detect and block third-party trackers disguising themselves as first-party " +
					"to circumvent browser privacy protections like ITP. Defaults to `false`.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"allow_affiliate": schema.BoolAttribute{
				MarkdownDescription: "Allow affiliate & tracking domains common on deals websites, in emails or in search results. " +
					"Those usually only get called after manually clicking on a link. " +
					"Your IP address will automatically be hidden from those websites to preserve your privacy. Defaults to `false`.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
		},
	}
}

func (r *PrivacyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PrivacyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// The privacy settings always exist, so creating means updating them.
	var data PrivacyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.update(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivacyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PrivacyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	privacy, err := r.client.GetPrivacy(ctx, data.ProfileID.ValueString())
	if nextdns.IsNotFound(err) {
		// The profile was deleted outside Terraform.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading NextDNS privacy settings", err.Error())
		return
	}

	blocklists, diags := listItemsToSet(ctx, privacy.Blocklists)
	resp.Diagnostics.Append(diags...)
	natives, diags := listItemsToSet(ctx, privacy.Natives)
	resp.Diagnostics.Append(diags...)

	data.Blocklists = blocklists
	data.Natives = natives
	data.DisguisedTrackers = types.BoolValue(privacy.DisguisedTrackers)
	data.AllowAffiliate = types.BoolValue(privacy.AllowAffiliate)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivacyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PrivacyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.update(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// update sends every setting in data to NextDNS.
func (r *PrivacyResource) update(ctx context.Context, data *PrivacyResourceModel) diag.Diagnostics {
	blocklists, diags := setToListItems(ctx, data.Blocklists)
	natives, nativeDiags := setToListItems(ctx, data.Natives)
	diags.Append(nativeDiags...)
	if diags.HasError() {
		return diags
	}

	privacy := &nextdns.Privacy{
		Blocklists:        blocklists,
		Natives:           natives,
		DisguisedTrackers: data.DisguisedTrackers.ValueBool(),
		AllowAffiliate:    data.AllowAffiliate.ValueBool(),
	}
	if err := r.client.UpdatePrivacy(ctx, data.ProfileID.ValueString(), privacy); err != nil {
		diags.AddError("Error updating NextDNS privacy settings", err.Error())
	}
	return diags
}

func (r *PrivacyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Intentionally empty: the privacy settings cannot be deleted, and
	// resetting them would silently remove blocklists. Terraform removes the
	// resource from the state after this returns.
}

func (r *PrivacyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("profile_id"), req, resp)
}
