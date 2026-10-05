package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jfgmesquita/terraform-provider-nextdns/internal/nextdns"
)

// Ensure SetupDataSource satisfies the framework interfaces.
var _ datasource.DataSource = &SetupDataSource{}

func NewSetupDataSource() datasource.DataSource {
	return &SetupDataSource{}
}

// SetupDataSource reads the information needed to use a NextDNS profile.
type SetupDataSource struct {
	client *nextdns.Client
}

// SetupDataSourceModel maps the data source schema to Go values.
type SetupDataSourceModel struct {
	ProfileID           types.String `tfsdk:"profile_id"`
	IPv4                []string     `tfsdk:"ipv4"`
	IPv6                []string     `tfsdk:"ipv6"`
	LinkedIPServers     []string     `tfsdk:"linked_ip_servers"`
	LinkedIP            types.String `tfsdk:"linked_ip"`
	LinkedIPDDNS        types.String `tfsdk:"linked_ip_ddns"`
	LinkedIPUpdateToken types.String `tfsdk:"linked_ip_update_token"`
	DNSCrypt            types.String `tfsdk:"dnscrypt"`
	DoHURL              types.String `tfsdk:"doh_url"`
	DoTHostname         types.String `tfsdk:"dot_hostname"`
}

func (d *SetupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_setup"
}

func (d *SetupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	stringList := func(description string) schema.ListAttribute {
		return schema.ListAttribute{MarkdownDescription: description, ElementType: types.StringType, Computed: true}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "The endpoints and DNS servers of a NextDNS profile. " +
			"Use them to configure routers, firewalls or devices with a profile managed by Terraform.\n\n" +
			"Endpoints: set up NextDNS with this profile using one of the endpoints (`dot_hostname`, `doh_url` or `ipv6`).\n\n" +
			"Linked IP: if you are unable to set up NextDNS using the NextDNS apps, DNS-over-TLS, DNS-over-HTTPS or IPv6, " +
			"then use the DNS servers in `linked_ip_servers` and link your IP. " +
			"This is mostly for use on home networks and not recommended on mobile.",

		Attributes: map[string]schema.Attribute{
			"profile_id": schema.StringAttribute{
				MarkdownDescription: "ID of the profile.",
				Required:            true,
			},
			"ipv4": stringList("IPv4 DNS servers dedicated to this profile. Usually empty: for IPv4, use `linked_ip_servers`."),
			"ipv6": stringList("IPv6 endpoint: the IPv6 DNS servers of this profile."),
			"linked_ip_servers": stringList("Linked IP DNS servers: IPv4 DNS servers that identify this profile by its linked IP, " +
				"the public IP address queries come from."),
			"linked_ip": schema.StringAttribute{
				MarkdownDescription: "Linked IP: the public IP address linked to this profile. Null if no IP is linked.",
				Computed:            true,
			},
			"linked_ip_ddns": schema.StringAttribute{
				MarkdownDescription: "The DDNS hostname whose IP address is linked to this profile. Null if none is set.",
				Computed:            true,
			},
			"linked_ip_update_token": schema.StringAttribute{
				MarkdownDescription: "Token to update the linked IP, for example from a router on a dynamic IP address. " +
					"It is a secret: anyone with it can change the linked IP. Like every value Terraform reads, " +
					"it is stored in the Terraform state, so keep the state secure.",
				Computed:  true,
				Sensitive: true,
			},
			"dnscrypt": schema.StringAttribute{
				MarkdownDescription: "DNSCrypt stamp of this profile.",
				Computed:            true,
			},
			"doh_url": schema.StringAttribute{
				MarkdownDescription: "DNS-over-HTTPS endpoint of this profile: `https://dns.nextdns.io/<profile ID>`.",
				Computed:            true,
			},
			"dot_hostname": schema.StringAttribute{
				MarkdownDescription: "DNS-over-TLS/QUIC endpoint of this profile: `<profile ID>.dns.nextdns.io`.",
				Computed:            true,
			},
		},
	}
}

func (d *SetupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*nextdns.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *nextdns.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *SetupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SetupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID := data.ProfileID.ValueString()
	setup, err := d.client.GetSetup(ctx, profileID)
	if nextdns.IsNotFound(err) {
		resp.Diagnostics.AddError("NextDNS profile not found", fmt.Sprintf("No profile with ID %q exists.", profileID))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading NextDNS setup", err.Error())
		return
	}

	data.IPv4 = setup.IPv4
	data.IPv6 = setup.IPv6
	data.LinkedIPServers = setup.LinkedIP.Servers
	data.LinkedIP = types.StringPointerValue(setup.LinkedIP.IP)
	data.LinkedIPDDNS = types.StringPointerValue(setup.LinkedIP.DDNS)
	data.LinkedIPUpdateToken = types.StringValue(setup.LinkedIP.UpdateToken)
	data.DNSCrypt = types.StringValue(setup.DNSCrypt)
	data.DoHURL = types.StringValue("https://dns.nextdns.io/" + profileID)
	data.DoTHostname = types.StringValue(profileID + ".dns.nextdns.io")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
