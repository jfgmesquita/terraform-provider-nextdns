package nextdns

import (
	"context"
	"net/http"
)

// ParentalControl holds a profile's parental control settings, except the
// recreation schedule, which is updated separately (see UpdateRecreation).
type ParentalControl struct {
	SafeSearch            bool                   `json:"safeSearch"`
	YouTubeRestrictedMode bool                   `json:"youtubeRestrictedMode"`
	BlockBypass           bool                   `json:"blockBypass"`
	Services              []ParentalControlEntry `json:"services"`
	Categories            []ParentalControlEntry `json:"categories"`
	// Recreation is only read: UpdateParentalControl never sends it.
	Recreation *Recreation `json:"recreation,omitempty"`
}

// ParentalControlEntry is one blocked service or category.
type ParentalControlEntry struct {
	ID     string `json:"id"`
	Active bool   `json:"active"`
	// Recreation allows the entry during recreation time.
	Recreation bool `json:"recreation"`
}

// Recreation is the weekly schedule during which entries marked with
// Recreation are not blocked. Times maps a lowercase weekday ("monday") to a
// time range. NextDNS stores times as "HH:MM:SS".
type Recreation struct {
	Times    map[string]RecreationTime `json:"times"`
	Timezone string                    `json:"timezone,omitempty"`
}

// RecreationTime is a time range within one day.
type RecreationTime struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// GetParentalControl returns the parental control settings of a profile.
func (c *Client) GetParentalControl(ctx context.Context, profileID string) (*ParentalControl, error) {
	var pc ParentalControl
	if err := c.Do(ctx, http.MethodGet, profilePath(profileID)+"/parentalControl", nil, &pc); err != nil {
		return nil, err
	}
	return &pc, nil
}

// UpdateParentalControl replaces the parental control settings of a profile,
// except the recreation schedule. NextDNS fails (HTTP 500 after almost a
// minute) when the schedule is sent together with services and categories,
// so the schedule has its own request.
func (c *Client) UpdateParentalControl(ctx context.Context, profileID string, pc *ParentalControl) error {
	body := *pc
	body.Recreation = nil
	if body.Services == nil {
		body.Services = []ParentalControlEntry{}
	}
	if body.Categories == nil {
		body.Categories = []ParentalControlEntry{}
	}
	return c.Do(ctx, http.MethodPatch, profilePath(profileID)+"/parentalControl", body, nil)
}

// UpdateRecreation replaces the whole recreation schedule of a profile. An
// empty Times removes the schedule.
func (c *Client) UpdateRecreation(ctx context.Context, profileID string, recreation Recreation) error {
	if recreation.Times == nil {
		recreation.Times = map[string]RecreationTime{}
	}
	body := map[string]Recreation{"recreation": recreation}
	return c.Do(ctx, http.MethodPatch, profilePath(profileID)+"/parentalControl", body, nil)
}
