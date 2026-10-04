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

// Ensure RewritesResource satisfies the framework interfaces.
var _ resource.Resource = &RewritesResource{}
var _ resource.ResourceWithImportState = &RewritesResource{}

func NewRewritesResource() resource.Resource {
	return &RewritesResource{}
}

// RewritesResource manages a profile's rewrites.
type RewritesResource struct {
	client *nextdns.Client
}

// RewritesResourceModel maps the resource schema to Go values.
type RewritesResourceModel struct {
	ProfileID types.String   `tfsdk:"profile_id"`
	Rewrites  []RewriteModel `tfsdk:"rewrite"`
}

// RewriteModel is one rewrite block.
type RewriteModel struct {
	Domain types.String `tfsdk:"domain"`
	Answer types.String `tfsdk:"answer"`
}

// rewriteKey identifies a rewrite: the same domain can have several answers.
type rewriteKey struct {
	domain string
	answer string
}

func (r *RewritesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rewrites"
}

func (r *RewritesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The rewrites of a NextDNS profile. Set or override the DNS response for any domain. " +
			"Rewrites apply to subdomains as well, and local IP addresses are supported as answers.\n\n" +
			"This resource manages all rewrites of the profile: rewrites added outside Terraform are removed on the next apply.\n\n" +
			"Removing this resource from the configuration, or destroying it, does not change anything in NextDNS: " +
			"Terraform only stops managing the rewrites. To remove rewrites, delete their blocks and apply.",

		Attributes: map[string]schema.Attribute{
			"profile_id": schema.StringAttribute{
				MarkdownDescription: "ID of the profile these rewrites belong to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"rewrite": schema.SetNestedBlock{
				MarkdownDescription: "A rewrite. Repeat the block for each rewrite. The same domain can have several answers.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"domain": schema.StringAttribute{
							MarkdownDescription: "The domain to rewrite, for example `router.home`. Its subdomains are rewritten too.",
							Required:            true,
						},
						"answer": schema.StringAttribute{
							MarkdownDescription: "The answer: an IPv4 address, an IPv6 address or a domain name, " +
								"for example `192.168.1.1`. NextDNS picks the record type (A, AAAA or CNAME) from it.",
							Required: true,
						},
					},
				},
			},
		},
	}
}

func (r *RewritesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RewritesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RewritesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.sync(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error updating NextDNS rewrites", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RewritesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RewritesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rewrites, err := r.client.ListRewrites(ctx, data.ProfileID.ValueString())
	if nextdns.IsNotFound(err) {
		// The profile was deleted outside Terraform.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading NextDNS rewrites", err.Error())
		return
	}

	// A set cannot hold the same element twice, so skip exact duplicates.
	seen := map[rewriteKey]bool{}
	data.Rewrites = make([]RewriteModel, 0, len(rewrites))
	for _, rw := range rewrites {
		key := rewriteKey{rw.Name, rw.Content}
		if seen[key] {
			continue
		}
		seen[key] = true
		data.Rewrites = append(data.Rewrites, RewriteModel{
			Domain: types.StringValue(rw.Name),
			Answer: types.StringValue(rw.Content),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RewritesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RewritesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.sync(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error updating NextDNS rewrites", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// sync makes the rewrites in NextDNS match data. The API cannot replace the
// whole list, so it compares with what NextDNS has now and deletes and adds
// rewrites one by one. Comparing with NextDNS, not with the Terraform state,
// means a later apply finishes any work an interrupted one left undone.
func (r *RewritesResource) sync(ctx context.Context, data *RewritesResourceModel) error {
	profileID := data.ProfileID.ValueString()

	current, err := r.client.ListRewrites(ctx, profileID)
	if err != nil {
		return err
	}

	desired := make([]rewriteKey, 0, len(data.Rewrites))
	for _, rw := range data.Rewrites {
		desired = append(desired, rewriteKey{rw.Domain.ValueString(), rw.Answer.ValueString()})
	}

	toDelete, toCreate := diffRewrites(current, desired)
	for _, id := range toDelete {
		if err := r.client.DeleteRewrite(ctx, profileID, id); err != nil && !nextdns.IsNotFound(err) {
			return err
		}
	}
	for _, key := range toCreate {
		if err := r.client.CreateRewrite(ctx, profileID, key.domain, key.answer); err != nil {
			return fmt.Errorf("adding %s → %s: %w", key.domain, key.answer, err)
		}
	}
	return nil
}

// diffRewrites returns the IDs of current rewrites that are not desired, or
// are duplicates, and the desired rewrites that do not exist yet.
func diffRewrites(current []nextdns.Rewrite, desired []rewriteKey) (toDelete []string, toCreate []rewriteKey) {
	want := map[rewriteKey]bool{}
	for _, key := range desired {
		want[key] = true
	}

	have := map[rewriteKey]bool{}
	for _, rw := range current {
		key := rewriteKey{rw.Name, rw.Content}
		if !want[key] || have[key] {
			toDelete = append(toDelete, rw.ID)
			continue
		}
		have[key] = true
	}

	for _, key := range desired {
		if !have[key] {
			toCreate = append(toCreate, key)
			have[key] = true
		}
	}
	return toDelete, toCreate
}

func (r *RewritesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Intentionally empty, like the other list resources: removing the
	// resource must not silently delete rewrites. Terraform removes the
	// resource from the state after this returns.
}

func (r *RewritesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("profile_id"), req, resp)
}
