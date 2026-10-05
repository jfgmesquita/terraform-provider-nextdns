package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccSetupDataSource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProfileResourceConfig(name) + `
data "nextdns_setup" "test" {
  profile_id = nextdns_profile.test.id
}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.nextdns_setup.test", tfjsonpath.New("ipv6"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue("data.nextdns_setup.test", tfjsonpath.New("linked_ip_servers"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue("data.nextdns_setup.test", tfjsonpath.New("linked_ip"), knownvalue.Null()),
					statecheck.ExpectKnownValue("data.nextdns_setup.test", tfjsonpath.New("linked_ip_update_token"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("data.nextdns_setup.test", tfjsonpath.New("doh_url"),
						knownvalue.StringRegexp(regexp.MustCompile(`^https://dns\.nextdns\.io/[0-9a-f]+$`))),
					statecheck.ExpectKnownValue("data.nextdns_setup.test", tfjsonpath.New("dot_hostname"),
						knownvalue.StringRegexp(regexp.MustCompile(`^[0-9a-f]+\.dns\.nextdns\.io$`))),
				},
			},
		},
	})
}

func TestAccSetupDataSourceMissingProfile(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "nextdns_setup" "test" {
  profile_id = "zzzzzz"
}
`,
				ExpectError: regexp.MustCompile(`NextDNS profile not found`),
			},
		},
	})
}
