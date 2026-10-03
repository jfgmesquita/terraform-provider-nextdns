package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccProfileResource(t *testing.T) {
	// A random name, so test runs never clash with each other or real profiles.
	name := acctest.RandomWithPrefix("tf-acc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read.
			{
				Config: testAccProfileResourceConfig(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nextdns_profile.test", tfjsonpath.New("name"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue("nextdns_profile.test", tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("nextdns_profile.test", tfjsonpath.New("fingerprint"), knownvalue.NotNull()),
				},
			},
			// Import by ID gives the same state.
			{
				ResourceName:      "nextdns_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Rename.
			{
				Config: testAccProfileResourceConfig(name + "-renamed"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nextdns_profile.test", tfjsonpath.New("name"), knownvalue.StringExact(name+"-renamed")),
				},
			},
			// Terraform deletes the profile automatically at the end.
		},
	})
}

func testAccProfileResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "nextdns_profile" "test" {
  name = %q
}
`, name)
}
