package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jfgmesquita/terraform-provider-nextdns/internal/nextdns"
)

// Ensure DomainListResource satisfies the framework interfaces.
var _ resource.Resource = &DomainListResource{}
var _ resource.ResourceWithImportState = &DomainListResource{}

// NewDenylistResource returns the nextdns_denylist resource.
func NewDenylistResource() resource.Resource {
	return &DomainListResource{
		list:        "denylist",
		description: "Denying a domain will automatically deny all its subdomains.",
	}
}

// NewAllowlistResource returns the nextdns_allowlist resource.
func NewAllowlistResource() resource.Resource {
	return &DomainListResource{
		list: "allowlist",
		description: "Allowing a domain will automatically allow all its subdomains. " +
			"Allowing takes precedence over everything else, including security features.",
	}
}

// DomainListResource manages a profile's denylist or allowlist. Both work the
// same way, so one implementation serves both.
type DomainListResource struct {
	client *nextdns.Client
	// list is the API path and resource name suffix: "denylist" or "allowlist".
	list        string
	description string
}

// DomainListResourceModel maps the resource schema to Go values.
type DomainListResourceModel struct {
	ProfileID types.String           `tfsdk:"profile_id"`
	Domains   []DomainListEntryModel `tfsdk:"domain"`
}

// DomainListEntryModel is one domain block.
type DomainListEntryModel struct {
	ID     types.String `tfsdk:"id"`
	Active types.Bool   `tfsdk:"active"`
}

func (r *DomainListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.list
}

func (r *DomainListResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf("The %s of a NextDNS profile. %s\n\n", r.list, r.description) +
			fmt.Sprintf("This resource manages the whole %s: domains added outside Terraform are removed on the next apply.\n\n", r.list) +
			"Removing this resource from the configuration, or destroying it, does not change anything in NextDNS: " +
			"Terraform only stops managing it. To remove domains, delete their blocks and apply.",

		Attributes: map[string]schema.Attribute{
			"profile_id": schema.StringAttribute{
				MarkdownDescription: "ID of the profile this list belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"domain": schema.SetNestedBlock{
				MarkdownDescription: "A domain in the " + r.list + ". Repeat the block for each domain.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The domain, for example `example.com`.",
							Required:            true,
						},
						"active": schema.BoolAttribute{
							MarkdownDescription: "Whether this entry is active. Inactive entries stay in the list but have no effect. Defaults to `true`.",
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(true),
						},
					},
				},
			},
		},
	}
}

func (r *DomainListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DomainListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// The list always exists, so creating means replacing it.
	var data DomainListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.replace(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error updating NextDNS "+r.list, err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DomainListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DomainListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entries, err := r.client.GetDomainList(ctx, data.ProfileID.ValueString(), r.list)
	if nextdns.IsNotFound(err) {
		// The profile was deleted outside Terraform.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading NextDNS "+r.list, err.Error())
		return
	}

	data.Domains = make([]DomainListEntryModel, 0, len(entries))
	for _, e := range entries {
		data.Domains = append(data.Domains, DomainListEntryModel{
			ID:     types.StringValue(e.ID),
			Active: types.BoolValue(e.Active),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DomainListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DomainListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.replace(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error updating NextDNS "+r.list, err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// replace sends the whole list in data to NextDNS.
func (r *DomainListResource) replace(ctx context.Context, data *DomainListResourceModel) error {
	entries := make([]nextdns.DomainListEntry, 0, len(data.Domains))
	for _, d := range data.Domains {
		entries = append(entries, nextdns.DomainListEntry{
			ID:     d.ID.ValueString(),
			Active: d.Active.ValueBool(),
		})
	}
	return r.client.ReplaceDomainList(ctx, data.ProfileID.ValueString(), r.list, entries)
}

func (r *DomainListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Intentionally empty, like the settings resources: removing the resource
	// must not silently empty the list. Terraform removes the resource from
	// the state after this returns.
}

func (r *DomainListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("profile_id"), req, resp)
}
