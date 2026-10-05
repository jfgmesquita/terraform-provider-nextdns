package nextdns

import (
	"context"
	"net/http"
)

// Setup holds the information needed to use a profile: its DNS servers and
// endpoints.
type Setup struct {
	IPv4     []string      `json:"ipv4"`
	IPv6     []string      `json:"ipv6"`
	LinkedIP SetupLinkedIP `json:"linkedIp"`
	// DNSCrypt is a DNS stamp ("sdns://...") of the DNS-over-HTTPS endpoint,
	// despite its name in the API.
	DNSCrypt string `json:"dnscrypt"`
}

// SetupLinkedIP holds the linked IP details. NextDNS identifies IPv4 queries
// to the linked IP servers by the public IP linked to the profile.
type SetupLinkedIP struct {
	Servers []string `json:"servers"`
	// IP and DDNS are null until an IP or a DDNS hostname is linked.
	IP   *string `json:"ip"`
	DDNS *string `json:"ddns"`
	// UpdateToken is a secret: it lets anyone change the linked IP.
	UpdateToken string `json:"updateToken"`
}

// GetSetup returns the setup information of a profile.
func (c *Client) GetSetup(ctx context.Context, profileID string) (*Setup, error) {
	var setup Setup
	if err := c.Do(ctx, http.MethodGet, profilePath(profileID)+"/setup", nil, &setup); err != nil {
		return nil, err
	}
	return &setup, nil
}
