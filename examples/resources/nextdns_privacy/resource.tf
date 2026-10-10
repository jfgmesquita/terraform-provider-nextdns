resource "nextdns_profile" "office" {
  name = "Office"
}

resource "nextdns_privacy" "office" {
  profile_id = nextdns_profile.office.id

  # IDs from https://api.nextdns.io/privacy/blocklists
  blocklists = ["nextdns-recommended", "oisd"]

  # IDs from https://api.nextdns.io/privacy/natives
  natives = ["apple", "windows"]

  disguised_trackers = true
  allow_affiliate    = false
}
