package nextdns

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient returns a client that talks to a fake NextDNS server, which
// answers every request with the given status code and body.
func newTestClient(t *testing.T, status int, body string, check func(r *http.Request)) *Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)

	client := NewClient("test-key")
	client.BaseURL = server.URL
	return client
}

func TestDoSendsRequest(t *testing.T) {
	client := newTestClient(t, http.StatusOK, `{"data":{}}`, func(r *http.Request) {
		if got := r.Header.Get("X-Api-Key"); got != "test-key" {
			t.Errorf("X-Api-Key = %q, want %q", got, "test-key")
		}
		if r.Method != http.MethodPatch || r.URL.Path != "/profiles/abc123" {
			t.Errorf("request = %s %s, want PATCH /profiles/abc123", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"name":"Home"}` {
			t.Errorf("body = %s", body)
		}
	})

	err := client.Do(t.Context(), http.MethodPatch, "/profiles/abc123", map[string]string{"name": "Home"}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDoDecodesData(t *testing.T) {
	client := newTestClient(t, http.StatusOK, `{"data":{"id":"abc123","name":"Home"}}`, nil)

	var profile struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := client.Do(t.Context(), http.MethodGet, "/profiles/abc123", nil, &profile); err != nil {
		t.Fatal(err)
	}
	if profile.ID != "abc123" || profile.Name != "Home" {
		t.Errorf("profile = %+v", profile)
	}
}

func TestDoNoContent(t *testing.T) {
	client := newTestClient(t, http.StatusNoContent, "", nil)

	var out struct{}
	if err := client.Do(t.Context(), http.MethodDelete, "/profiles/abc123", nil, &out); err != nil {
		t.Fatal(err)
	}
}

func TestDoErrors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantMsg string
	}{
		{
			// Issues #1 and #2: a 5xx body must be parsed, never discarded.
			name:    "server error with errors list",
			status:  http.StatusInternalServerError,
			body:    `{"errors":[{"code":"internalServerError"}]}`,
			wantMsg: "NextDNS API error (HTTP 500): internalServerError",
		},
		{
			name:    "server error without JSON",
			status:  http.StatusBadGateway,
			body:    "Bad Gateway\n",
			wantMsg: "NextDNS API error (HTTP 502): Bad Gateway",
		},
		{
			name:    "client error with detail and parameter",
			status:  http.StatusBadRequest,
			body:    `{"errors":[{"code":"invalid","detail":"Invalid TLD","source":{"parameter":"id"}}]}`,
			wantMsg: "NextDNS API error (HTTP 400): invalid: Invalid TLD (parameter: id)",
		},
		{
			// Response recorded from the real API.
			name:    "client error with pointer",
			status:  http.StatusBadRequest,
			body:    `{"errors":[{"code":"enum","source":{"pointer":"/logs/retention"},"detail":"` + "`/logs/retention`" + ` must be equal to one of the allowed values."}]}`,
			wantMsg: "NextDNS API error (HTTP 400): enum: `/logs/retention` must be equal to one of the allowed values. (at /logs/retention)",
		},
		{
			// NextDNS sometimes reports errors with HTTP 200.
			name:    "errors with HTTP 200",
			status:  http.StatusOK,
			body:    `{"errors":[{"code":"duplicate"}]}`,
			wantMsg: "NextDNS API error (HTTP 200): duplicate",
		},
		{
			name:    "several errors",
			status:  http.StatusBadRequest,
			body:    `{"errors":[{"code":"invalid"},{"code":"required"}]}`,
			wantMsg: "NextDNS API error (HTTP 400): invalid; required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, tt.status, tt.body, nil)

			err := client.Do(t.Context(), http.MethodGet, "/profiles", nil, nil)

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error = %v, want *APIError", err)
			}
			if apiErr.StatusCode != tt.status {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.status)
			}
			if err.Error() != tt.wantMsg {
				t.Errorf("message = %q, want %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestDoRefusesRedirectToAnotherHost(t *testing.T) {
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("other host received a request (X-Api-Key = %q)", r.Header.Get("X-Api-Key"))
	}))
	t.Cleanup(other.Close)

	redirecting := httptest.NewServer(http.RedirectHandler(other.URL+"/profiles", http.StatusFound))
	t.Cleanup(redirecting.Close)

	client := NewClient("test-key")
	client.BaseURL = redirecting.URL

	err := client.Do(t.Context(), http.MethodGet, "/profiles", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "refusing redirect") {
		t.Fatalf("error = %v, want a refused redirect", err)
	}
}

func TestDoFollowsRedirectOnSameHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/old" {
			http.Redirect(w, r, "/new", http.StatusFound)
			return
		}
		_, _ = io.WriteString(w, `{"data":{"name":"Home"}}`)
	}))
	t.Cleanup(server.Close)

	client := NewClient("test-key")
	client.BaseURL = server.URL

	var out struct {
		Name string `json:"name"`
	}
	if err := client.Do(t.Context(), http.MethodGet, "/old", nil, &out); err != nil {
		t.Fatal(err)
	}
	if out.Name != "Home" {
		t.Errorf("name = %q, want %q", out.Name, "Home")
	}
}
