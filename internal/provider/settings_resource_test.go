package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/jfgmesquita/terraform-provider-nextdns/internal/nextdns"
)

func TestAccSettingsResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc")
	var profileID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// A retention typo fails at plan time, before anything is created (issue #10).
			{
				Config: testAccSettingsResourceConfig(name, `
  logs_retention = "7 days"
`),
				ExpectError: regexp.MustCompile(`logs_retention`),
			},
			// Set some settings.
			{
				Config: testAccSettingsResourceConfig(name, `
  logs_enabled    = true
  logs_client_ips = false
  logs_retention  = "1 week"
  logs_location   = "eu"
  web3            = true
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nextdns_settings.test", tfjsonpath.New("logs_retention"), knownvalue.StringExact("1 week")),
					statecheck.ExpectKnownValue("nextdns_settings.test", tfjsonpath.New("logs_location"), knownvalue.StringExact("eu")),
					statecheck.ExpectKnownValue("nextdns_settings.test", tfjsonpath.New("block_page"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("nextdns_settings.test", tfjsonpath.New("logs_client_ips"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("nextdns_settings.test", tfjsonpath.New("logs_domains"), knownvalue.Bool(true)),
				},
				Check: func(s *terraform.State) error {
					profileID = s.RootModule().Resources["nextdns_profile.test"].Primary.ID
					return nil
				},
			},
			// Import by profile ID gives the same state.
			{
				ResourceName:                         "nextdns_settings.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "profile_id",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources["nextdns_settings.test"].Primary.Attributes["profile_id"], nil
				},
			},
			// Change settings: settings left out go back to their defaults.
			{
				Config: testAccSettingsResourceConfig(name, `
  logs_enabled            = true
  block_page              = true
  cache_boost             = true
  bypass_age_verification = true
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nextdns_settings.test", tfjsonpath.New("logs_retention"), knownvalue.StringExact("3 months")),
					statecheck.ExpectKnownValue("nextdns_settings.test", tfjsonpath.New("logs_client_ips"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("nextdns_settings.test", tfjsonpath.New("web3"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("nextdns_settings.test", tfjsonpath.New("bypass_age_verification"), knownvalue.Bool(true)),
				},
			},
			// Removing the resource must not change NextDNS (issue #6).
			{
				Config: testAccProfileResourceConfig(name),
				Check: func(*terraform.State) error {
					client := nextdns.NewClient(os.Getenv("NEXTDNS_API_KEY"))
					settings, err := client.GetSettings(context.Background(), profileID)
					if err != nil {
						return err
					}
					if !settings.Logs.Enabled || !settings.BlockPage.Enabled || !settings.BAV {
						return errors.New("settings changed after removing nextdns_settings from the configuration")
					}
					return nil
				},
			},
		},
	})
}

// TestAccSettingsResourceAllValues checks that NextDNS accepts every log
// retention and location this provider allows.
func TestAccSettingsResourceAllValues(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc")

	var steps []resource.TestStep
	for i, ret := range logRetentions {
		location := logLocations[i%len(logLocations)]
		steps = append(steps, resource.TestStep{
			Config: testAccSettingsResourceConfig(name, fmt.Sprintf(`
  logs_retention = %q
  logs_location  = %q
`, ret.name, location)),
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("nextdns_settings.test", tfjsonpath.New("logs_retention"), knownvalue.StringExact(ret.name)),
				statecheck.ExpectKnownValue("nextdns_settings.test", tfjsonpath.New("logs_location"), knownvalue.StringExact(location)),
			},
		})
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    steps,
	})
}

func testAccSettingsResourceConfig(name, settings string) string {
	return testAccProfileResourceConfig(name) + fmt.Sprintf(`
resource "nextdns_settings" "test" {
  profile_id = nextdns_profile.test.id
%s}
`, settings)
}
