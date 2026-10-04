package provider

import (
	"context"
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

func TestAccDenylistResource(t *testing.T)  { testAccDomainListResource(t, "denylist") }
func TestAccAllowlistResource(t *testing.T) { testAccDomainListResource(t, "allowlist") }

func testAccDomainListResource(t *testing.T, list string) {
	name := acctest.RandomWithPrefix("tf-acc")
	address := "nextdns_" + list + ".test"
	var profileID string

	domain := func(id string, active bool) knownvalue.Check {
		return knownvalue.ObjectExact(map[string]knownvalue.Check{
			"id":     knownvalue.StringExact(id),
			"active": knownvalue.Bool(active),
		})
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Two domains, one inactive. NextDNS reorders them, which a set ignores.
			{
				Config: testAccDomainListResourceConfig(name, list, `
  domain {
    id = "b.example.com"
  }
  domain {
    id     = "a.example.com"
    active = false
  }
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(address, tfjsonpath.New("domain"), knownvalue.SetExact([]knownvalue.Check{
						domain("a.example.com", false),
						domain("b.example.com", true),
					})),
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
			// Activate one, remove one, add one.
			{
				Config: testAccDomainListResourceConfig(name, list, `
  domain {
    id = "a.example.com"
  }
  domain {
    id = "c.example.com"
  }
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(address, tfjsonpath.New("domain"), knownvalue.SetExact([]knownvalue.Check{
						domain("a.example.com", true),
						domain("c.example.com", true),
					})),
				},
			},
			// No blocks empties the list.
			{
				Config: testAccDomainListResourceConfig(name, list, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(address, tfjsonpath.New("domain"), knownvalue.SetSizeExact(0)),
				},
			},
			// Add one back, to check it survives the next step.
			{
				Config: testAccDomainListResourceConfig(name, list, `
  domain {
    id = "d.example.com"
  }
`),
			},
			// Removing the resource must not change NextDNS (issue #6).
			{
				Config: testAccProfileResourceConfig(name),
				Check: func(*terraform.State) error {
					client := nextdns.NewClient(os.Getenv("NEXTDNS_API_KEY"))
					entries, err := client.GetDomainList(context.Background(), profileID, list)
					if err != nil {
						return err
					}
					if len(entries) != 1 || entries[0].ID != "d.example.com" {
						return fmt.Errorf("%s changed after removing the resource from the configuration: %+v", list, entries)
					}
					return nil
				},
			},
		},
	})
}

func testAccDomainListResourceConfig(name, list, domains string) string {
	return testAccProfileResourceConfig(name) + fmt.Sprintf(`
resource "nextdns_%s" "test" {
  profile_id = nextdns_profile.test.id
%s}
`, list, domains)
}
