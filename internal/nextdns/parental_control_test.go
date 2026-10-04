package nextdns

import (
	"io"
	"net/http"
	"testing"
)

func TestGetParentalControl(t *testing.T) {
	// Response recorded from the real API.
	body := `{"data":{"safeSearch":true,"youtubeRestrictedMode":true,"blockBypass":false,` +
		`"services":[{"id":"tiktok","website":"https://www.tiktok.com","recreation":true,"active":true}],` +
		`"categories":[{"id":"gambling","recreation":false,"active":true}],` +
		`"recreation":{"times":{"tuesday":{"start":"18:00:00","end":"20:00:00"}},"timezone":"Europe/Lisbon"}}}`
	client := newTestClient(t, http.StatusOK, body, nil)

	pc, err := client.GetParentalControl(t.Context(), "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if !pc.SafeSearch || len(pc.Services) != 1 || !pc.Services[0].Recreation || pc.Categories[0].ID != "gambling" {
		t.Errorf("parental control = %+v", pc)
	}
	if pc.Recreation.Timezone != "Europe/Lisbon" || pc.Recreation.Times["tuesday"].Start != "18:00:00" {
		t.Errorf("recreation = %+v", pc.Recreation)
	}
}

func TestUpdateParentalControlNeverSendsRecreation(t *testing.T) {
	client := newTestClient(t, http.StatusNoContent, "", func(r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		want := `{"safeSearch":true,"youtubeRestrictedMode":false,"blockBypass":false,"services":[],"categories":[]}`
		if string(body) != want {
			t.Errorf("body = %s, want %s", body, want)
		}
	})

	pc := &ParentalControl{
		SafeSearch: true,
		Recreation: &Recreation{Timezone: "Europe/Lisbon"},
	}
	if err := client.UpdateParentalControl(t.Context(), "abc123", pc); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateRecreationEmpty(t *testing.T) {
	client := newTestClient(t, http.StatusNoContent, "", func(r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"recreation":{"times":{}}}` {
			t.Errorf("body = %s", body)
		}
	})

	if err := client.UpdateRecreation(t.Context(), "abc123", Recreation{}); err != nil {
		t.Fatal(err)
	}
}
