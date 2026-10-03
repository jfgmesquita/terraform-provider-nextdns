package nextdns

import (
	"context"
	"net/http"
)

// Security holds a profile's security settings.
type Security struct {
	ThreatIntelligenceFeeds  bool          `json:"threatIntelligenceFeeds"`
	AIThreatDetection        bool          `json:"aiThreatDetection"`
	GoogleSafeBrowsing       bool          `json:"googleSafeBrowsing"`
	Cryptojacking            bool          `json:"cryptojacking"`
	DNSRebinding             bool          `json:"dnsRebinding"`
	IDNHomographs            bool          `json:"idnHomographs"`
	Typosquatting            bool          `json:"typosquatting"`
	DGA                      bool          `json:"dga"`
	NRD                      bool          `json:"nrd"`
	NewlyActiveDomains       bool          `json:"newlyActiveDomains"`
	FreeHostingDomains       bool          `json:"freeHostingDomains"`
	DDNS                     bool          `json:"ddns"`
	TunnelingEndpoints       bool          `json:"tunnelingEndpoints"`
	DataDropServices         bool          `json:"dataDropServices"`
	ResidentialHosting       bool          `json:"residentialHosting"`
	UntrustedCertificates    bool          `json:"untrustedCertificates"`
	DNSPayloadDelivery       bool          `json:"dnsPayloadDelivery"`
	DecentralizedWebGateways bool          `json:"decentralizedWebGateways"`
	HighRiskTLDs             bool          `json:"highRiskTlds"`
	Parking                  bool          `json:"parking"`
	CSAM                     bool          `json:"csam"`
	TLDs                     []SecurityTLD `json:"tlds"`
}

// SecurityTLD is a top-level domain blocked by the profile.
type SecurityTLD struct {
	ID string `json:"id"`
}

// GetSecurity returns the security settings of a profile.
func (c *Client) GetSecurity(ctx context.Context, profileID string) (*Security, error) {
	var security Security
	if err := c.Do(ctx, http.MethodGet, profilePath(profileID)+"/security", nil, &security); err != nil {
		return nil, err
	}
	return &security, nil
}

// UpdateSecurity replaces all security settings of a profile, including the
// TLD list. Unlike PUT /security/tlds, an invalid TLD here leaves the
// existing settings unchanged.
func (c *Client) UpdateSecurity(ctx context.Context, profileID string, security *Security) error {
	return c.Do(ctx, http.MethodPatch, profilePath(profileID)+"/security", security, nil)
}
