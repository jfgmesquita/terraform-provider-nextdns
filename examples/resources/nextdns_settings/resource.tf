resource "nextdns_profile" "office" {
  name = "Office"
}

resource "nextdns_settings" "office" {
  profile_id = nextdns_profile.office.id

  logs_enabled    = true
  logs_client_ips = true
  logs_domains    = true
  logs_retention  = "1 month"
  logs_location   = "eu"

  block_page = true

  anonymized_ecs   = true
  cache_boost      = true
  cname_flattening = true

  bypass_age_verification = false

  web3 = false
}
