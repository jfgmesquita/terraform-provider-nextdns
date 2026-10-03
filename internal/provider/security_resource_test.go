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

func TestAccSecurityResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc")
	var profileID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Turn some settings on.
			{
				Config: testAccSecurityResourceConfig(name, `
  cryptojacking = true
  csam          = true
  tlds          = ["zip", "mov"]
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nextdns_security.test", tfjsonpath.New("cryptojacking"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("nextdns_security.test", tfjsonpath.New("dga"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("nextdns_security.test", tfjsonpath.New("tlds"), knownvalue.SetExact([]knownvalue.Check{
						knownvalue.StringExact("zip"),
						knownvalue.StringExact("mov"),
					})),
				},
				Check: func(s *terraform.State) error {
					profileID = s.RootModule().Resources["nextdns_profile.test"].Primary.ID
					return nil
				},
			},
			// Import by profile ID gives the same state.
			{
				ResourceName:                         "nextdns_security.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "profile_id",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources["nextdns_security.test"].Primary.Attributes["profile_id"], nil
				},
			},
			// Change settings: csam is no longer listed, so it is turned off.
			{
				Config: testAccSecurityResourceConfig(name, `
  cryptojacking = true
  dga           = true
  tlds          = ["zip"]
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nextdns_security.test", tfjsonpath.New("csam"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("nextdns_security.test", tfjsonpath.New("dga"), knownvalue.Bool(true)),
				},
			},
			// Removing the resource must not change NextDNS (issue #6).
			{
				Config: testAccProfileResourceConfig(name),
				Check: func(*terraform.State) error {
					client := nextdns.NewClient(os.Getenv("NEXTDNS_API_KEY"))
					security, err := client.GetSecurity(context.Background(), profileID)
					if err != nil {
						return err
					}
					if !security.Cryptojacking || !security.DGA || len(security.TLDs) != 1 {
						return errors.New("security settings changed after removing nextdns_security from the configuration")
					}
					return nil
				},
			},
		},
	})
}

func testAccSecurityResourceConfig(name, settings string) string {
	return testAccProfileResourceConfig(name) + fmt.Sprintf(`
resource "nextdns_security" "test" {
  profile_id = nextdns_profile.test.id
%s}
`, settings)
}
