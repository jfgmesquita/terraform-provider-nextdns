resource "nextdns_profile" "home" {
  name = "Home"
}

# Every setting is listed here. Any setting left out is set to its default.
resource "nextdns_settings" "home" {
  profile_id = nextdns_profile.home.id

  logs_enabled     = true
  logs_drop_ip     = false
  logs_drop_domain = false
  logs_retention   = "1 month"
  logs_location    = "eu"

  block_page = true

  anonymized_ecs   = true
  cache_boost      = true
  cname_flattening = true

  bypass_age_verification = false
  web3                    = false
}
