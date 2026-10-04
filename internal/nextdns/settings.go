package nextdns

import (
	"context"
	"net/http"
)

// Settings holds a profile's general settings.
type Settings struct {
	Logs        SettingsLogs        `json:"logs"`
	BlockPage   SettingsBlockPage   `json:"blockPage"`
	Performance SettingsPerformance `json:"performance"`
	// BAV is "bypass age verification".
	BAV  bool `json:"bav"`
	Web3 bool `json:"web3"`
}

// SettingsLogs holds the query log settings.
type SettingsLogs struct {
	Enabled bool `json:"enabled"`
	Drop    struct {
		IP     bool `json:"ip"`
		Domain bool `json:"domain"`
	} `json:"drop"`
	// Retention is in seconds.
	Retention int    `json:"retention"`
	Location  string `json:"location"`
}

// SettingsBlockPage holds the block page settings.
type SettingsBlockPage struct {
	Enabled bool `json:"enabled"`
}

// SettingsPerformance holds the performance settings.
type SettingsPerformance struct {
	ECS             bool `json:"ecs"`
	CacheBoost      bool `json:"cacheBoost"`
	CNAMEFlattening bool `json:"cnameFlattening"`
}

// GetSettings returns the general settings of a profile.
func (c *Client) GetSettings(ctx context.Context, profileID string) (*Settings, error) {
	var settings Settings
	if err := c.Do(ctx, http.MethodGet, profilePath(profileID)+"/settings", nil, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

// UpdateSettings replaces all general settings of a profile. An invalid value
// rejects the whole request and leaves the existing settings unchanged.
func (c *Client) UpdateSettings(ctx context.Context, profileID string, settings *Settings) error {
	return c.Do(ctx, http.MethodPatch, profilePath(profileID)+"/settings", settings, nil)
}
