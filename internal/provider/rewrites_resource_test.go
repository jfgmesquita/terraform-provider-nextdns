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

func TestAccRewritesResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc")
	var profileID string

	rewrite := func(domain, answer string) knownvalue.Check {
		return knownvalue.ObjectExact(map[string]knownvalue.Check{
			"domain": knownvalue.StringExact(domain),
			"answer": knownvalue.StringExact(answer),
		})
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// IPv4, IPv6 with two answers for one domain, and a domain name.
			{
				Config: testAccRewritesResourceConfig(name, `
  rewrite {
    domain = "router.home"
    answer = "192.168.1.1"
  }
  rewrite {
    domain = "v6.home"
    answer = "fd00::1"
  }
  rewrite {
    domain = "v6.home"
    answer = "fd00::2"
  }
  rewrite {
    domain = "www.example.com"
    answer = "example.org"
  }
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nextdns_rewrites.test", tfjsonpath.New("rewrite"), knownvalue.SetSizeExact(4)),
				},
				Check: func(s *terraform.State) error {
					profileID = s.RootModule().Resources["nextdns_profile.test"].Primary.ID
					return nil
				},
			},
			// Import by profile ID gives the same state.
			{
				ResourceName:                         "nextdns_rewrites.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "profile_id",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources["nextdns_rewrites.test"].Primary.Attributes["profile_id"], nil
				},
			},
			// Change an answer, drop some, keep one.
			{
				Config: testAccRewritesResourceConfig(name, `
  rewrite {
    domain = "router.home"
    answer = "192.168.1.254"
  }
  rewrite {
    domain = "v6.home"
    answer = "fd00::1"
  }
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nextdns_rewrites.test", tfjsonpath.New("rewrite"), knownvalue.SetExact([]knownvalue.Check{
						rewrite("router.home", "192.168.1.254"),
						rewrite("v6.home", "fd00::1"),
					})),
				},
			},
			// No blocks removes every rewrite.
			{
				Config: testAccRewritesResourceConfig(name, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nextdns_rewrites.test", tfjsonpath.New("rewrite"), knownvalue.SetSizeExact(0)),
				},
			},
			// Add one back, to check it survives the next step.
			{
				Config: testAccRewritesResourceConfig(name, `
  rewrite {
    domain = "nas.home"
    answer = "192.168.1.10"
  }
`),
			},
			// Removing the resource must not change NextDNS (issue #6).
			{
				Config: testAccProfileResourceConfig(name),
				Check: func(*terraform.State) error {
					client := nextdns.NewClient(os.Getenv("NEXTDNS_API_KEY"))
					rewrites, err := client.ListRewrites(context.Background(), profileID)
					if err != nil {
						return err
					}
					if len(rewrites) != 1 || rewrites[0].Name != "nas.home" {
						return fmt.Errorf("rewrites changed after removing the resource from the configuration: %+v", rewrites)
					}
					return nil
				},
			},
		},
	})
}

func testAccRewritesResourceConfig(name, rewrites string) string {
	return testAccProfileResourceConfig(name) + fmt.Sprintf(`
resource "nextdns_rewrites" "test" {
  profile_id = nextdns_profile.test.id
%s}
`, rewrites)
}
