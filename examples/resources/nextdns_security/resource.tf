resource "nextdns_profile" "office" {
  name = "Office"
}

resource "nextdns_security" "office" {
  profile_id = nextdns_profile.office.id

  threat_intelligence_feeds  = true
  ai_threat_detection        = true
  google_safe_browsing       = true
  nrd                        = true
  newly_active_domains       = true
  free_hosting_domains       = true
  ddns                       = true
  tunneling_endpoints        = true
  residential_hosting        = true
  untrusted_certificates     = true
  data_drop_services         = false
  dns_payload_delivery       = true
  dns_rebinding              = true
  cryptojacking              = true
  decentralized_web_gateways = true
  parking                    = true
  high_risk_tlds             = true
  dga                        = true
  idn_homographs             = true
  typosquatting              = true
  csam                       = true

  tlds = ["zip", "mov"]
}
