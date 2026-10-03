package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/jfgmesquita/terraform-provider-nextdns/internal/nextdns"
)

func TestAccPrivacyResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc")
	var profileID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Set lists and one setting.
			{
				Config: testAccPrivacyResourceConfig(name, `
  blocklists         = ["nextdns-recommended", "oisd"]
  natives            = ["apple"]
  disguised_trackers = true
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nextdns_privacy.test", tfjsonpath.New("blocklists"), knownvalue.SetExact([]knownvalue.Check{
						knownvalue.StringExact("nextdns-recommended"),
						knownvalue.StringExact("oisd"),
					})),
					statecheck.ExpectKnownValue("nextdns_privacy.test", tfjsonpath.New("disguised_trackers"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("nextdns_privacy.test", tfjsonpath.New("allow_affiliate"), knownvalue.Bool(false)),
				},
				Check: func(s *terraform.State) error {
					profileID = s.RootModule().Resources["nextdns_profile.test"].Primary.ID
					return nil
				},
			},
			// Import by profile ID gives the same state.
			{
				ResourceName:                         "nextdns_privacy.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "profile_id",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources["nextdns_privacy.test"].Primary.Attributes["profile_id"], nil
				},
			},
			// Change lists and settings.
			{
				Config: testAccPrivacyResourceConfig(name, `
  blocklists      = ["oisd"]
  natives         = ["apple", "windows"]
  allow_affiliate = true
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nextdns_privacy.test", tfjsonpath.New("blocklists"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue("nextdns_privacy.test", tfjsonpath.New("natives"), knownvalue.SetSizeExact(2)),
					statecheck.ExpectKnownValue("nextdns_privacy.test", tfjsonpath.New("disguised_trackers"), knownvalue.Bool(false)),
				},
			},
			// Removing the list lines empties the lists.
			{
				Config: testAccPrivacyResourceConfig(name, `
  allow_affiliate = true
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nextdns_privacy.test", tfjsonpath.New("blocklists"), knownvalue.SetSizeExact(0)),
					statecheck.ExpectKnownValue("nextdns_privacy.test", tfjsonpath.New("natives"), knownvalue.SetSizeExact(0)),
				},
			},
			// Removing the resource must not change NextDNS (issue #6).
			{
				Config: testAccProfileResourceConfig(name),
				Check: func(*terraform.State) error {
					client := nextdns.NewClient(os.Getenv("NEXTDNS_API_KEY"))
					privacy, err := client.GetPrivacy(context.Background(), profileID)
					if err != nil {
						return err
					}
					if !privacy.AllowAffiliate {
						return errors.New("privacy settings changed after removing nextdns_privacy from the configuration")
					}
					return nil
				},
			},
		},
	})
}

func testAccPrivacyResourceConfig(name, settings string) string {
	return testAccProfileResourceConfig(name) + fmt.Sprintf(`
resource "nextdns_privacy" "test" {
  profile_id = nextdns_profile.test.id
%s}
`, settings)
}
