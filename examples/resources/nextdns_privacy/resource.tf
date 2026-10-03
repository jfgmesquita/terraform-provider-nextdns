resource "nextdns_profile" "home" {
  name = "Home"
}

# Every setting is listed here. Any setting left out is turned off,
# and any list left out is emptied.
resource "nextdns_privacy" "home" {
  profile_id = nextdns_profile.home.id

  # IDs from https://api.nextdns.io/privacy/blocklists
  blocklists = ["nextdns-recommended", "oisd"]

  # IDs from https://api.nextdns.io/privacy/natives
  natives = ["apple", "windows"]

  disguised_trackers = true
  allow_affiliate    = false
}
