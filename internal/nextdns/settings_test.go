package nextdns

import (
	"net/http"
	"testing"
)

func TestGetSettings(t *testing.T) {
	// Response recorded from the real API.
	body := `{"data":{"logs":{"enabled":false,"drop":{"ip":true,"domain":false},"retention":604800,"location":"eu"},` +
		`"blockPage":{"enabled":false},"performance":{"ecs":false,"cacheBoost":false,"cnameFlattening":false},"bav":false,"web3":false}}`
	client := newTestClient(t, http.StatusOK, body, nil)

	settings, err := client.GetSettings(t.Context(), "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if !settings.Logs.Drop.IP || settings.Logs.Retention != 604800 || settings.Logs.Location != "eu" {
		t.Errorf("logs = %+v", settings.Logs)
	}
}
