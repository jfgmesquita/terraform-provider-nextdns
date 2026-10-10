package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jfgmesquita/terraform-provider-nextdns/internal/nextdns"
)

// Ensure SettingsResource satisfies the framework interfaces.
var _ resource.Resource = &SettingsResource{}
var _ resource.ResourceWithImportState = &SettingsResource{}

func NewSettingsResource() resource.Resource {
	return &SettingsResource{}
}

// SettingsResource manages the general settings of a NextDNS profile.
type SettingsResource struct {
	client *nextdns.Client
}

// SettingsResourceModel maps the resource schema to Go values.
type SettingsResourceModel struct {
	ProfileID             types.String `tfsdk:"profile_id"`
	LogsEnabled           types.Bool   `tfsdk:"logs_enabled"`
	LogsClientIPs         types.Bool   `tfsdk:"logs_client_ips"`
	LogsDomains           types.Bool   `tfsdk:"logs_domains"`
	LogsRetention         types.String `tfsdk:"logs_retention"`
	LogsLocation          types.String `tfsdk:"logs_location"`
	BlockPage             types.Bool   `tfsdk:"block_page"`
	AnonymizedECS         types.Bool   `tfsdk:"anonymized_ecs"`
	CacheBoost            types.Bool   `tfsdk:"cache_boost"`
	CNAMEFlattening       types.Bool   `tfsdk:"cname_flattening"`
	BypassAgeVerification types.Bool   `tfsdk:"bypass_age_verification"`
	Web3                  types.Bool   `tfsdk:"web3"`
}

// logRetentions lists the accepted log retention periods, in the order shown
// in the dashboard, with their value in seconds.
var logRetentions = []struct {
	name    string
	seconds int
}{
	{"1 hour", 3600},
	{"6 hours", 21600},
	{"1 day", 86400},
	{"1 week", 604800},
	{"1 month", 2592000},
	{"3 months", 7776000},
	{"6 months", 15552000},
	{"1 year", 31536000},
	{"2 years", 63072000},
}

// logLocations lists the accepted log storage locations.
var logLocations = []string{"us", "eu", "ch"}

func (r *SettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_settings"
}

// boolSetting returns an optional true/false attribute that defaults to false.
func boolSetting(description string) schema.BoolAttribute {
	return schema.BoolAttribute{
		MarkdownDescription: description + " Defaults to `false`.",
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(false),
	}
}

func (r *SettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	retentionNames := make([]string, 0, len(logRetentions))
	for _, ret := range logRetentions {
		retentionNames = append(retentionNames, ret.name)
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "The general settings of a NextDNS profile: logs, block page, performance, " +
			"bypass age verification and Web3. The profile name is set in `nextdns_profile`.\n\n" +
			"~> Removing this resource from the configuration does not change the settings in NextDNS: " +
			"Terraform only stops managing them. To reset a setting to its default, remove it from the resource.",

		Attributes: map[string]schema.Attribute{
			"profile_id": schema.StringAttribute{
				MarkdownDescription: "ID of the profile these settings belong to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"logs_enabled": boolSetting("Log the DNS queries of this profile."),
			"logs_client_ips": schema.BoolAttribute{
				MarkdownDescription: "Log client IP addresses. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"logs_domains": schema.BoolAttribute{
				MarkdownDescription: "Log domains. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"logs_retention": schema.StringAttribute{
				MarkdownDescription: "How long logs are kept: one of `" + strings.Join(retentionNames, "`, `") + "`. Defaults to `3 months`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("3 months"),
				Validators:          []validator.String{stringvalidator.OneOf(retentionNames...)},
			},
			"logs_location": schema.StringAttribute{
				MarkdownDescription: "Where logs are stored: `us` (United States), `eu` (European Union) " +
					"or `ch` (Switzerland). Defaults to `us`.",
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString("us"),
				Validators: []validator.String{stringvalidator.OneOf(logLocations...)},
			},
			"block_page": boolSetting("Display a block page when a domain is being blocked. This may slightly increase page load time " +
				"and an HTTPS warning may appear in some cases. When disabled, blocked queries will be answered with the " +
				"unspecified address (0.0.0.0 or ::)."),
			"anonymized_ecs": boolSetting("Speed up the delivery of data from content delivery " +
				"networks without exposing your IP address."),
			"cache_boost": boolSetting("Minimize DNS queries by enforcing a minimum TTL (Time to live)."),
			"cname_flattening": boolSetting("Prevent CNAME-chasing resolvers from making unnecessary queries and polluting " +
				"the logs with intermediate domains."),
			"bypass_age_verification": boolSetting("Automatically bypass age verification checks used by certain websites, " +
				"such as adult content sites, to verify a visitor’s age before allowing access. By enabling this feature, " +
				"you acknowledge that you are legally old enough to access the content."),
			"web3": boolSetting("Web3 refers to a decentralized and censorship-resistant online ecosystem comprised of innovative " +
				"technologies such as blockchain-based domain registries (e.g., Ethereum Name Service) and distributed content " +
				"storage and delivery networks (e.g., IPFS). When enabled, NextDNS will act as an unfiltered gateway to this new " +
				"Web, letting you experience it firsthand without the need to install anything. As most browsers only support " +
				"classic top-level domains at the moment, you should add a trailing slash (\"/\") when trying to access a Web3 " +
				"domain directly (e.g., \"vitalik.eth/\" instead of \"vitalik.eth\")."),
		},
	}
}

func (r *SettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// The settings always exist, so creating means updating them.
	var data SettingsResourceModel
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

func (r *SettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, err := r.client.GetSettings(ctx, data.ProfileID.ValueString())
	if nextdns.IsNotFound(err) {
		// The profile was deleted outside Terraform.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading NextDNS settings", err.Error())
		return
	}

	retention := ""
	for _, ret := range logRetentions {
		if ret.seconds == settings.Logs.Retention {
			retention = ret.name
		}
	}
	if retention == "" {
		resp.Diagnostics.AddError("Unexpected NextDNS log retention",
			fmt.Sprintf("NextDNS returned a log retention of %d seconds, which this provider does not know. "+
				"Please report this issue to the provider developers.", settings.Logs.Retention))
		return
	}

	data.LogsEnabled = types.BoolValue(settings.Logs.Enabled)
	// The API stores what is dropped; the dashboard and this resource show what is logged.
	data.LogsClientIPs = types.BoolValue(!settings.Logs.Drop.IP)
	data.LogsDomains = types.BoolValue(!settings.Logs.Drop.Domain)
	data.LogsRetention = types.StringValue(retention)
	data.LogsLocation = types.StringValue(settings.Logs.Location)
	data.BlockPage = types.BoolValue(settings.BlockPage.Enabled)
	data.AnonymizedECS = types.BoolValue(settings.Performance.ECS)
	data.CacheBoost = types.BoolValue(settings.Performance.CacheBoost)
	data.CNAMEFlattening = types.BoolValue(settings.Performance.CNAMEFlattening)
	data.BypassAgeVerification = types.BoolValue(settings.BAV)
	data.Web3 = types.BoolValue(settings.Web3)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SettingsResourceModel
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
func (r *SettingsResource) update(ctx context.Context, data *SettingsResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	settings := &nextdns.Settings{}
	settings.Logs.Enabled = data.LogsEnabled.ValueBool()
	settings.Logs.Drop.IP = !data.LogsClientIPs.ValueBool()
	settings.Logs.Drop.Domain = !data.LogsDomains.ValueBool()
	settings.Logs.Location = data.LogsLocation.ValueString()
	for _, ret := range logRetentions {
		if ret.name == data.LogsRetention.ValueString() {
			settings.Logs.Retention = ret.seconds
		}
	}
	settings.BlockPage.Enabled = data.BlockPage.ValueBool()
	settings.Performance.ECS = data.AnonymizedECS.ValueBool()
	settings.Performance.CacheBoost = data.CacheBoost.ValueBool()
	settings.Performance.CNAMEFlattening = data.CNAMEFlattening.ValueBool()
	settings.BAV = data.BypassAgeVerification.ValueBool()
	settings.Web3 = data.Web3.ValueBool()

	if err := r.client.UpdateSettings(ctx, data.ProfileID.ValueString(), settings); err != nil {
		diags.AddError("Error updating NextDNS settings", err.Error())
	}
	return diags
}

func (r *SettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Intentionally empty: the settings cannot be deleted, and resetting them
	// could, for example, turn logs off. Terraform removes the resource from
	// the state after this returns.
}

func (r *SettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("profile_id"), req, resp)
}
