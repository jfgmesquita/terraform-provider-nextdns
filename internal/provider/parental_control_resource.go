package provider

import (
	"context"
	"fmt"
	"regexp"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jfgmesquita/terraform-provider-nextdns/internal/nextdns"
)

// Ensure ParentalControlResource satisfies the framework interfaces.
var _ resource.Resource = &ParentalControlResource{}
var _ resource.ResourceWithImportState = &ParentalControlResource{}
var _ resource.ResourceWithValidateConfig = &ParentalControlResource{}

func NewParentalControlResource() resource.Resource {
	return &ParentalControlResource{}
}

// ParentalControlResource manages the parental control settings of a NextDNS profile.
type ParentalControlResource struct {
	client *nextdns.Client
}

// ParentalControlResourceModel maps the resource schema to Go values.
type ParentalControlResourceModel struct {
	ProfileID             types.String                         `tfsdk:"profile_id"`
	SafeSearch            types.Bool                           `tfsdk:"safe_search"`
	YouTubeRestrictedMode types.Bool                           `tfsdk:"youtube_restricted_mode"`
	BlockBypass           types.Bool                           `tfsdk:"block_bypass"`
	Services              map[string]ParentalControlEntryModel `tfsdk:"services"`
	Categories            map[string]ParentalControlEntryModel `tfsdk:"categories"`
	Recreation            *RecreationModel                     `tfsdk:"recreation"`
}

// ParentalControlEntryModel is one service or category, keyed by its ID.
// A map rather than a set of blocks: with a set, the framework applies the
// defaults of both flags when only one of them is set.
type ParentalControlEntryModel struct {
	Active     types.Bool `tfsdk:"active"`
	Recreation types.Bool `tfsdk:"recreation"`
}

// RecreationModel is the recreation block. A nil day has no recreation time.
type RecreationModel struct {
	Timezone  types.String         `tfsdk:"timezone"`
	Monday    *RecreationTimeModel `tfsdk:"monday"`
	Tuesday   *RecreationTimeModel `tfsdk:"tuesday"`
	Wednesday *RecreationTimeModel `tfsdk:"wednesday"`
	Thursday  *RecreationTimeModel `tfsdk:"thursday"`
	Friday    *RecreationTimeModel `tfsdk:"friday"`
	Saturday  *RecreationTimeModel `tfsdk:"saturday"`
	Sunday    *RecreationTimeModel `tfsdk:"sunday"`
}

// RecreationTimeModel is the recreation time of one day.
type RecreationTimeModel struct {
	Start types.String `tfsdk:"start"`
	End   types.String `tfsdk:"end"`
}

// weekdays links each weekday's name in the API to its field in the model.
func (m *RecreationModel) weekdays() []struct {
	name string
	time **RecreationTimeModel
} {
	return []struct {
		name string
		time **RecreationTimeModel
	}{
		{"monday", &m.Monday},
		{"tuesday", &m.Tuesday},
		{"wednesday", &m.Wednesday},
		{"thursday", &m.Thursday},
		{"friday", &m.Friday},
		{"saturday", &m.Saturday},
		{"sunday", &m.Sunday},
	}
}

// timeOfDay matches "HH:MM", for example "18:00".
var timeOfDay = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

func (r *ParentalControlResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_parental_control"
}

// entryAttributeTypes is the type of one service or category entry.
var entryAttributeTypes = map[string]attr.Type{"active": types.BoolType, "recreation": types.BoolType}

func entryMap(kind, description, example, catalog string) schema.MapNestedAttribute {
	return schema.MapNestedAttribute{
		MarkdownDescription: description + " Map keys are " + kind + " IDs, for example `" + example + "`; " +
			"the available IDs are listed at " + catalog + ". Use `{}` for the defaults, for example `" + example + " = {}`.",
		Optional: true,
		Computed: true,
		Default: mapdefault.StaticValue(types.MapValueMust(
			types.ObjectType{AttrTypes: entryAttributeTypes}, map[string]attr.Value{})),
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"active": schema.BoolAttribute{
					MarkdownDescription: "Whether this entry is active. Inactive entries stay in the list but have no effect. Defaults to `true`.",
					Optional:            true,
					Computed:            true,
					Default:             booldefault.StaticBool(true),
				},
				"recreation": schema.BoolAttribute{
					MarkdownDescription: "Do not block this " + kind + " during recreation time. Defaults to `false`.",
					Optional:            true,
					Computed:            true,
					Default:             booldefault.StaticBool(false),
				},
			},
		},
	}
}

func (r *ParentalControlResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	timeAttribute := func(day string) schema.SingleNestedAttribute {
		hhmm := []validator.String{stringvalidator.RegexMatches(timeOfDay, "must be a time in the format HH:MM, for example 18:00")}
		return schema.SingleNestedAttribute{
			MarkdownDescription: "Recreation time on " + day + ", for example `{ start = \"18:00\", end = \"20:00\" }`.",
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"start": schema.StringAttribute{
					MarkdownDescription: "Start time, in the format `HH:MM`.",
					Required:            true,
					Validators:          hhmm,
				},
				"end": schema.StringAttribute{
					MarkdownDescription: "End time, in the format `HH:MM`.",
					Required:            true,
					Validators:          hhmm,
				},
			},
		}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "The parental control settings of a NextDNS profile.\n\n" +
			"This resource manages all parental control settings: any setting left out of the configuration is turned off, " +
			"and any service, category or recreation time left out is removed.\n\n" +
			"Removing this resource from the configuration, or destroying it, does not change any settings in NextDNS: " +
			"Terraform only stops managing them. To turn settings off, set them to `false` or remove them, and apply.\n\n" +
			profilePartNotes("nextdns_parental_control"),

		Attributes: map[string]schema.Attribute{
			"profile_id": schema.StringAttribute{
				MarkdownDescription: "ID of the profile these settings belong to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"safe_search": boolSetting("SafeSearch: filter explicit results on all major search engines, including images " +
				"and videos. This will also block access to search engines not supporting this feature."),
			"youtube_restricted_mode": boolSetting("YouTube Restricted Mode: filter out mature videos on YouTube and block " +
				"embedded mature videos from being watched on other websites. This will also hide all comments."),
			"block_bypass": boolSetting("Block bypass methods: prevent or hinder the use of methods that can help bypass " +
				"NextDNS filtering on the network. This includes VPNs, proxies, Tor-related software and encrypted DNS providers."),
			"services": entryMap("service", "Websites, apps & games: restrict access to specific websites, apps and games.",
				"tiktok", "https://api.nextdns.io/parentalcontrol/services"),
			"categories": entryMap("category", "Categories: restrict access to specific categories of websites and apps.",
				"gambling", "https://api.nextdns.io/parentalcontrol/categories"),
		},
		Blocks: map[string]schema.Block{
			"recreation": schema.SingleNestedBlock{
				MarkdownDescription: "Recreation time: set a period for each day of the week during which the services " +
					"and categories with `recreation = true` will not be blocked — e.g., allow Facebook on Mondays and Tuesdays " +
					"between 6pm and 8pm. Leave out a day to have no recreation time on that day.",
				Attributes: map[string]schema.Attribute{
					"timezone": schema.StringAttribute{
						MarkdownDescription: "Timezone of the recreation times, for example `Europe/Lisbon`. Required when the block is set.",
						Optional:            true,
					},
					"monday":    timeAttribute("Monday"),
					"tuesday":   timeAttribute("Tuesday"),
					"wednesday": timeAttribute("Wednesday"),
					"thursday":  timeAttribute("Thursday"),
					"friday":    timeAttribute("Friday"),
					"saturday":  timeAttribute("Saturday"),
					"sunday":    timeAttribute("Sunday"),
				},
			},
		},
	}
}

// ValidateConfig checks the recreation block: NextDNS drops the timezone of a
// schedule without days, so a block needs a timezone and at least one day.
func (r *ParentalControlResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ParentalControlResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.Recreation == nil {
		return
	}

	if data.Recreation.Timezone.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("recreation").AtName("timezone"),
			"Missing recreation timezone", "The recreation block needs a timezone, for example \"Europe/Lisbon\".")
	}

	days := 0
	for _, day := range data.Recreation.weekdays() {
		if *day.time != nil {
			days++
		}
	}
	if days == 0 {
		resp.Diagnostics.AddAttributeError(path.Root("recreation"),
			"Missing recreation time", "The recreation block needs at least one day. To remove recreation time, remove the block.")
	}
}

func (r *ParentalControlResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ParentalControlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// The settings always exist, so creating means updating them.
	var data ParentalControlResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.update(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error updating NextDNS parental control", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ParentalControlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ParentalControlResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pc, err := r.client.GetParentalControl(ctx, data.ProfileID.ValueString())
	if nextdns.IsNotFound(err) {
		// The profile was deleted outside Terraform.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading NextDNS parental control", err.Error())
		return
	}

	data.SafeSearch = types.BoolValue(pc.SafeSearch)
	data.YouTubeRestrictedMode = types.BoolValue(pc.YouTubeRestrictedMode)
	data.BlockBypass = types.BoolValue(pc.BlockBypass)
	data.Services = entriesFromAPI(pc.Services)
	data.Categories = entriesFromAPI(pc.Categories)

	data.Recreation = nil
	if pc.Recreation != nil && len(pc.Recreation.Times) > 0 {
		recreation := &RecreationModel{Timezone: types.StringValue(pc.Recreation.Timezone)}
		for _, day := range recreation.weekdays() {
			if t, ok := pc.Recreation.Times[day.name]; ok {
				// NextDNS stores "HH:MM:SS"; the configuration uses "HH:MM".
				*day.time = &RecreationTimeModel{
					Start: types.StringValue(hhmm(t.Start)),
					End:   types.StringValue(hhmm(t.End)),
				}
			}
		}
		data.Recreation = recreation
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ParentalControlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ParentalControlResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.update(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Error updating NextDNS parental control", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// update sends every setting in data to NextDNS. NextDNS fails when the
// recreation schedule is sent together with the services and categories, so
// the schedule has its own request. To fail closed, it uses three requests:
//
//  1. settings, services and categories, with no recreation allowed;
//  2. the recreation schedule;
//  3. services and categories again, with their recreation flags.
//
// If a request fails, NextDNS is left stricter than configured, never more
// permissive: entries never get recreation time under an outdated schedule.
func (r *ParentalControlResource) update(ctx context.Context, data *ParentalControlResourceModel) error {
	profileID := data.ProfileID.ValueString()

	pc := &nextdns.ParentalControl{
		SafeSearch:            data.SafeSearch.ValueBool(),
		YouTubeRestrictedMode: data.YouTubeRestrictedMode.ValueBool(),
		BlockBypass:           data.BlockBypass.ValueBool(),
		Services:              entriesToAPI(data.Services),
		Categories:            entriesToAPI(data.Categories),
	}

	strict := *pc
	strict.Services = withoutRecreation(pc.Services)
	strict.Categories = withoutRecreation(pc.Categories)
	if err := r.client.UpdateParentalControl(ctx, profileID, &strict); err != nil {
		return err
	}

	recreation := nextdns.Recreation{Times: map[string]nextdns.RecreationTime{}}
	if data.Recreation != nil {
		recreation.Timezone = data.Recreation.Timezone.ValueString()
		for _, day := range data.Recreation.weekdays() {
			if t := *day.time; t != nil {
				recreation.Times[day.name] = nextdns.RecreationTime{
					Start: t.Start.ValueString() + ":00",
					End:   t.End.ValueString() + ":00",
				}
			}
		}
	}
	if err := r.client.UpdateRecreation(ctx, profileID, recreation); err != nil {
		return fmt.Errorf("updating recreation time: %w", err)
	}

	if err := r.client.UpdateParentalControl(ctx, profileID, pc); err != nil {
		return fmt.Errorf("allowing recreation time: %w", err)
	}
	return nil
}

// withoutRecreation returns a copy of entries with recreation time turned off.
func withoutRecreation(entries []nextdns.ParentalControlEntry) []nextdns.ParentalControlEntry {
	out := make([]nextdns.ParentalControlEntry, len(entries))
	for i, e := range entries {
		e.Recreation = false
		out[i] = e
	}
	return out
}

func entriesToAPI(entries map[string]ParentalControlEntryModel) []nextdns.ParentalControlEntry {
	ids := make([]string, 0, len(entries))
	for id := range entries {
		ids = append(ids, id)
	}
	// Sorted, so requests are the same on every run.
	sort.Strings(ids)

	out := make([]nextdns.ParentalControlEntry, 0, len(entries))
	for _, id := range ids {
		out = append(out, nextdns.ParentalControlEntry{
			ID:         id,
			Active:     entries[id].Active.ValueBool(),
			Recreation: entries[id].Recreation.ValueBool(),
		})
	}
	return out
}

func entriesFromAPI(entries []nextdns.ParentalControlEntry) map[string]ParentalControlEntryModel {
	out := make(map[string]ParentalControlEntryModel, len(entries))
	for _, e := range entries {
		out[e.ID] = ParentalControlEntryModel{
			Active:     types.BoolValue(e.Active),
			Recreation: types.BoolValue(e.Recreation),
		}
	}
	return out
}

// hhmm turns "18:00:00" into "18:00".
func hhmm(t string) string {
	if len(t) > 5 {
		return t[:5]
	}
	return t
}

func (r *ParentalControlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Intentionally empty, like the other settings resources: removing the
	// resource must not silently turn parental control off. Terraform removes
	// the resource from the state after this returns.
}

func (r *ParentalControlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("profile_id"), req, resp)
}
