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

func TestAccParentalControlResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc")
	address := "nextdns_parental_control.test"
	var profileID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// A recreation block without days fails at plan time.
			{
				Config: testAccParentalControlResourceConfig(name, `
  recreation {
    timezone = "Europe/Lisbon"
  }
`),
				ExpectError: regexp.MustCompile(`Missing recreation time`),
			},
			// A badly formatted time fails at plan time.
			{
				Config: testAccParentalControlResourceConfig(name, `
  recreation {
    timezone = "Europe/Lisbon"
    monday   = { start = "6pm", end = "20:00" }
  }
`),
				ExpectError: regexp.MustCompile(`HH:MM`),
			},
			// Services, categories and recreation time in one apply: the old
			// provider always got HTTP 500 here (issue #4).
			{
				Config: testAccParentalControlResourceConfig(name, `
  safe_search             = true
  youtube_restricted_mode = true

  services = {
    tiktok   = { recreation = true }
    fortnite = { active = false }
  }
  categories = {
    gambling = {}
  }

  recreation {
    timezone = "Europe/Lisbon"
    monday   = { start = "18:00", end = "20:30" }
    saturday = { start = "10:00", end = "12:00" }
  }
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(address, tfjsonpath.New("services"), knownvalue.MapExact(map[string]knownvalue.Check{
						"tiktok": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"active": knownvalue.Bool(true), "recreation": knownvalue.Bool(true),
						}),
						"fortnite": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"active": knownvalue.Bool(false), "recreation": knownvalue.Bool(false),
						}),
					})),
					statecheck.ExpectKnownValue(address, tfjsonpath.New("categories").AtMapKey("gambling").AtMapKey("active"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(address, tfjsonpath.New("recreation").AtMapKey("monday").AtMapKey("end"), knownvalue.StringExact("20:30")),
					statecheck.ExpectKnownValue(address, tfjsonpath.New("block_bypass"), knownvalue.Bool(false)),
				},
				Check: func(s *terraform.State) error {
					profileID = s.RootModule().Resources["nextdns_profile.test"].Primary.ID
					return nil
				},
			},
			// Import by profile ID gives the same state.
			{
				ResourceName:                         address,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "profile_id",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources[address].Primary.Attributes["profile_id"], nil
				},
			},
			// Change days, lists and settings.
			{
				Config: testAccParentalControlResourceConfig(name, `
  block_bypass = true

  services = {
    tiktok = {}
  }
  categories = {
    gaming = { recreation = true }
  }

  recreation {
    timezone = "Europe/Lisbon"
    sunday   = { start = "15:00", end = "17:00" }
  }
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(address, tfjsonpath.New("safe_search"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(address, tfjsonpath.New("services"), knownvalue.MapSizeExact(1)),
					statecheck.ExpectKnownValue(address, tfjsonpath.New("recreation").AtMapKey("monday"), knownvalue.Null()),
					statecheck.ExpectKnownValue(address, tfjsonpath.New("recreation").AtMapKey("sunday").AtMapKey("start"), knownvalue.StringExact("15:00")),
				},
			},
			// Leaving out the lists and the recreation block removes them.
			{
				Config: testAccParentalControlResourceConfig(name, `
  block_bypass = true
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(address, tfjsonpath.New("services"), knownvalue.MapSizeExact(0)),
					statecheck.ExpectKnownValue(address, tfjsonpath.New("categories"), knownvalue.MapSizeExact(0)),
					statecheck.ExpectKnownValue(address, tfjsonpath.New("recreation"), knownvalue.Null()),
				},
			},
			// Removing the resource must not change NextDNS (issue #6).
			{
				Config: testAccProfileResourceConfig(name),
				Check: func(*terraform.State) error {
					client := nextdns.NewClient(os.Getenv("NEXTDNS_API_KEY"))
					pc, err := client.GetParentalControl(context.Background(), profileID)
					if err != nil {
						return err
					}
					if !pc.BlockBypass {
						return errors.New("parental control changed after removing the resource from the configuration")
					}
					return nil
				},
			},
		},
	})
}

func testAccParentalControlResourceConfig(name, settings string) string {
	return testAccProfileResourceConfig(name) + fmt.Sprintf(`
resource "nextdns_parental_control" "test" {
  profile_id = nextdns_profile.test.id
%s}
`, settings)
}
