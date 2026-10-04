package nextdns

import (
	"io"
	"net/http"
	"testing"
)

func TestListRewrites(t *testing.T) {
	// Response recorded from the real API.
	body := `{"data":[{"id":"r910xzju","name":"router.home","type":"A","content":"192.168.1.1"},` +
		`{"id":"5fvty956","name":"v6.home","type":"AAAA","content":"fd00::1"}]}`
	client := newTestClient(t, http.StatusOK, body, nil)

	rewrites, err := client.ListRewrites(t.Context(), "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if len(rewrites) != 2 || rewrites[1].ID != "5fvty956" || rewrites[1].Type != "AAAA" || rewrites[1].Content != "fd00::1" {
		t.Errorf("rewrites = %+v", rewrites)
	}
}

func TestCreateRewriteSendsOnlyNameAndContent(t *testing.T) {
	client := newTestClient(t, http.StatusOK, `{"data":{"id":"r910xzju"}}`, func(r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost || r.URL.Path != "/profiles/abc123/rewrites" ||
			string(body) != `{"name":"router.home","content":"192.168.1.1"}` {
			t.Errorf("request = %s %s %s", r.Method, r.URL.Path, body)
		}
	})

	if err := client.CreateRewrite(t.Context(), "abc123", "router.home", "192.168.1.1"); err != nil {
		t.Fatal(err)
	}
}
