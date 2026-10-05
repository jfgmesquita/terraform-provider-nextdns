package nextdns

import (
	"net/http"
	"testing"
)

func TestGetSetup(t *testing.T) {
	// Response recorded from the real API, with a fake update token.
	body := `{"data":{"ipv4":[],"ipv6":["2a07:a8c0::8c:b1e6","2a07:a8c1::8c:b1e6"],` +
		`"linkedIp":{"servers":["45.90.28.202","45.90.30.202"],"ip":null,"ddns":null,"updateToken":"fake-token"},` +
		`"dnscrypt":"sdns://AgEAAAAAAAAAAAAOZG5zLm5leHRkbnMuaW8HLzhjYjFlNg"}}`
	client := newTestClient(t, http.StatusOK, body, nil)

	setup, err := client.GetSetup(t.Context(), "8cb1e6")
	if err != nil {
		t.Fatal(err)
	}
	if len(setup.IPv4) != 0 || len(setup.IPv6) != 2 || setup.IPv6[0] != "2a07:a8c0::8c:b1e6" {
		t.Errorf("setup = %+v", setup)
	}
	if setup.LinkedIP.IP != nil || setup.LinkedIP.DDNS != nil || setup.LinkedIP.UpdateToken != "fake-token" {
		t.Errorf("linked ip = %+v", setup.LinkedIP)
	}
}
