package nextdns

import (
	"context"
	"net/http"
)

// Privacy holds a profile's privacy settings.
type Privacy struct {
	Blocklists        []ListItem `json:"blocklists"`
	Natives           []ListItem `json:"natives"`
	DisguisedTrackers bool       `json:"disguisedTrackers"`
	AllowAffiliate    bool       `json:"allowAffiliate"`
}

// GetPrivacy returns the privacy settings of a profile.
func (c *Client) GetPrivacy(ctx context.Context, profileID string) (*Privacy, error) {
	var privacy Privacy
	if err := c.Do(ctx, http.MethodGet, profilePath(profileID)+"/privacy", nil, &privacy); err != nil {
		return nil, err
	}
	return &privacy, nil
}

// UpdatePrivacy replaces all privacy settings of a profile. An invalid
// blocklist or native ID leaves the existing settings unchanged.
func (c *Client) UpdatePrivacy(ctx context.Context, profileID string, privacy *Privacy) error {
	return c.Do(ctx, http.MethodPatch, profilePath(profileID)+"/privacy", privacy, nil)
}
