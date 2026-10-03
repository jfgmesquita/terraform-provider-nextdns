package nextdns

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestUpdateSecuritySendsEmptyTLDList(t *testing.T) {
	client := newTestClient(t, http.StatusNoContent, "", func(r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/profiles/abc123/security" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatal(err)
		}
		// null would not clear the list: it must be an empty array.
		if tlds, ok := body["tlds"].([]any); !ok || len(tlds) != 0 {
			t.Errorf("tlds = %v, want []", body["tlds"])
		}
		if body["cryptojacking"] != true {
			t.Errorf("cryptojacking = %v, want true", body["cryptojacking"])
		}
	})

	security := &Security{Cryptojacking: true, TLDs: []ListItem{}}
	if err := client.UpdateSecurity(t.Context(), "abc123", security); err != nil {
		t.Fatal(err)
	}
}

func TestGetSecurity(t *testing.T) {
	body := `{"data":{"cryptojacking":true,"csam":false,"highRiskTlds":true,"tlds":[{"id":"zip"},{"id":"mov"}]}}`
	client := newTestClient(t, http.StatusOK, body, nil)

	security, err := client.GetSecurity(t.Context(), "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if !security.Cryptojacking || security.CSAM || !security.HighRiskTLDs {
		t.Errorf("security = %+v", security)
	}
	if len(security.TLDs) != 2 || security.TLDs[0].ID != "zip" || security.TLDs[1].ID != "mov" {
		t.Errorf("tlds = %+v", security.TLDs)
	}
}
