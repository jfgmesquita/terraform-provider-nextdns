package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories starts the provider for acceptance tests.
// Terraform calls the factory for each command it runs during a test.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"nextdns": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck skips acceptance tests when no NextDNS API key is available,
// for example in pull requests from forks, which do not receive repository secrets.
func testAccPreCheck(t *testing.T) {
	if os.Getenv("NEXTDNS_API_KEY") == "" {
		t.Skip("NEXTDNS_API_KEY is not set")
	}
}
