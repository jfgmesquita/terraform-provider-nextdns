package nextdns

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

// Profile is a NextDNS profile. Its settings (security, privacy, ...) are
// managed through their own endpoints.
type Profile struct {
	ID          string `json:"id"`
	Fingerprint string `json:"fingerprint"`
	Name        string `json:"name"`
}

func profilePath(id string) string {
	return "/profiles/" + url.PathEscape(id)
}

// CreateProfile creates a profile and returns it, including its new ID.
func (c *Client) CreateProfile(ctx context.Context, name string) (*Profile, error) {
	var profile Profile
	err := c.Do(ctx, http.MethodPost, "/profiles", map[string]string{"name": name}, &profile)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// GetProfile returns the profile with the given ID.
func (c *Client) GetProfile(ctx context.Context, id string) (*Profile, error) {
	var profile Profile
	if err := c.Do(ctx, http.MethodGet, profilePath(id), nil, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

// UpdateProfileName renames the profile with the given ID.
func (c *Client) UpdateProfileName(ctx context.Context, id, name string) error {
	return c.Do(ctx, http.MethodPatch, profilePath(id), map[string]string{"name": name}, nil)
}

// DeleteProfile deletes the profile with the given ID.
func (c *Client) DeleteProfile(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, profilePath(id), nil, nil)
}

// IsNotFound reports whether err means the requested object does not exist.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}
