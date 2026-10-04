package nextdns

import (
	"context"
	"net/http"
)

// DomainListEntry is one domain in a profile's denylist or allowlist.
type DomainListEntry struct {
	ID     string `json:"id"`
	Active bool   `json:"active"`
}

// GetDomainList returns a profile's domain list. list is "denylist" or
// "allowlist".
func (c *Client) GetDomainList(ctx context.Context, profileID, list string) ([]DomainListEntry, error) {
	var entries []DomainListEntry
	if err := c.Do(ctx, http.MethodGet, profilePath(profileID)+"/"+list, nil, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// ReplaceDomainList replaces a profile's whole domain list. list is
// "denylist" or "allowlist". An invalid or duplicate domain leaves the
// existing list unchanged.
func (c *Client) ReplaceDomainList(ctx context.Context, profileID, list string, entries []DomainListEntry) error {
	if entries == nil {
		// null would not clear the list: it must be an empty array.
		entries = []DomainListEntry{}
	}
	return c.Do(ctx, http.MethodPut, profilePath(profileID)+"/"+list, entries, nil)
}
