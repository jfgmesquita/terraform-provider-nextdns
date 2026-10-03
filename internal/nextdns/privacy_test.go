package nextdns

import (
	"net/http"
	"testing"
)

func TestGetPrivacy(t *testing.T) {
	// Shortened from a real API response: blocklists carry extra details,
	// which are ignored.
	body := `{"data":{"disguisedTrackers":true,"allowAffiliate":false,` +
		`"blocklists":[{"id":"oisd","name":"OISD","entries":243971,"updatedOn":"2026-10-03T23:15:14.000Z"}],` +
		`"natives":[{"id":"apple"}]}}`
	client := newTestClient(t, http.StatusOK, body, nil)

	privacy, err := client.GetPrivacy(t.Context(), "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if !privacy.DisguisedTrackers || privacy.AllowAffiliate {
		t.Errorf("privacy = %+v", privacy)
	}
	if len(privacy.Blocklists) != 1 || privacy.Blocklists[0].ID != "oisd" {
		t.Errorf("blocklists = %+v", privacy.Blocklists)
	}
	if len(privacy.Natives) != 1 || privacy.Natives[0].ID != "apple" {
		t.Errorf("natives = %+v", privacy.Natives)
	}
}
