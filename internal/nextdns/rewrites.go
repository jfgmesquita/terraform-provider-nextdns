package nextdns

import (
	"context"
	"net/http"
	"net/url"
)

// Rewrite overrides the DNS answer for a domain. NextDNS sets the ID and the
// record type (A, AAAA or CNAME) itself.
type Rewrite struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name"`
	Type    string `json:"type,omitempty"`
	Content string `json:"content"`
}

// ListRewrites returns a profile's rewrites.
func (c *Client) ListRewrites(ctx context.Context, profileID string) ([]Rewrite, error) {
	var rewrites []Rewrite
	if err := c.Do(ctx, http.MethodGet, profilePath(profileID)+"/rewrites", nil, &rewrites); err != nil {
		return nil, err
	}
	return rewrites, nil
}

// CreateRewrite adds one rewrite to a profile. The API cannot replace the
// whole list at once.
func (c *Client) CreateRewrite(ctx context.Context, profileID, name, content string) error {
	return c.Do(ctx, http.MethodPost, profilePath(profileID)+"/rewrites", Rewrite{Name: name, Content: content}, nil)
}

// DeleteRewrite removes one rewrite from a profile.
func (c *Client) DeleteRewrite(ctx context.Context, profileID, rewriteID string) error {
	return c.Do(ctx, http.MethodDelete, profilePath(profileID)+"/rewrites/"+url.PathEscape(rewriteID), nil, nil)
}
