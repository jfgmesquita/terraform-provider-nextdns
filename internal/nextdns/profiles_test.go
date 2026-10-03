package nextdns

import (
	"io"
	"net/http"
	"testing"
)

func TestCreateProfile(t *testing.T) {
	// Response recorded from the real API. Note that "fingerprint" repeats the
	// ID here: only GET /profiles/{id} returns the real fingerprint.
	body := `{"data":{"id":"fd5a74","fingerprint":"fd5a74","role":"owner","name":"Home"}}`
	client := newTestClient(t, http.StatusOK, body, func(r *http.Request) {
		got, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost || r.URL.Path != "/profiles" || string(got) != `{"name":"Home"}` {
			t.Errorf("request = %s %s %s", r.Method, r.URL.Path, got)
		}
	})

	profile, err := client.CreateProfile(t.Context(), "Home")
	if err != nil {
		t.Fatal(err)
	}
	if profile.ID != "fd5a74" || profile.Fingerprint != "fd5a74" || profile.Name != "Home" {
		t.Errorf("profile = %+v", profile)
	}
}

func TestGetProfileNotFound(t *testing.T) {
	// Response recorded from the real API for a deleted profile.
	client := newTestClient(t, http.StatusNotFound, `{"errors":[{"code":"notFound"}]}`, nil)

	_, err := client.GetProfile(t.Context(), "fd5a74")
	if !IsNotFound(err) {
		t.Errorf("IsNotFound(%v) = false, want true", err)
	}
}

func TestIsNotFoundOtherErrors(t *testing.T) {
	client := newTestClient(t, http.StatusInternalServerError, `{"errors":[{"code":"internalServerError"}]}`, nil)

	_, err := client.GetProfile(t.Context(), "fd5a74")
	if err == nil || IsNotFound(err) {
		t.Errorf("IsNotFound(%v) = true, want false", err)
	}
}
