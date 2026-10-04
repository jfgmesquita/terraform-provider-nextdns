package nextdns

import (
	"io"
	"net/http"
	"testing"
)

func TestReplaceDomainListSendsEmptyArray(t *testing.T) {
	client := newTestClient(t, http.StatusNoContent, "", func(r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPut || r.URL.Path != "/profiles/abc123/denylist" || string(body) != "[]" {
			t.Errorf("request = %s %s %s", r.Method, r.URL.Path, body)
		}
	})

	if err := client.ReplaceDomainList(t.Context(), "abc123", "denylist", nil); err != nil {
		t.Fatal(err)
	}
}

func TestReplaceDomainListInvalid(t *testing.T) {
	// Response recorded from the real API: errors come with HTTP 200.
	client := newTestClient(t, http.StatusOK, `{"errors":[{"code":"invalid"}]}`, nil)

	err := client.ReplaceDomainList(t.Context(), "abc123", "denylist", []DomainListEntry{{ID: "not a domain", Active: true}})
	if err == nil || err.Error() != "NextDNS API error (HTTP 200): invalid" {
		t.Errorf("error = %v", err)
	}
}
