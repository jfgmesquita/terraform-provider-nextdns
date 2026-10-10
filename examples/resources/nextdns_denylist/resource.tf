resource "nextdns_profile" "office" {
  name = "Office"
}

resource "nextdns_denylist" "office" {
  profile_id = nextdns_profile.office.id

  domain {
    id = "example.com"
  }

  domain {
    id = "example.org"
  }

  domain {
    id     = "example.net"
    active = false
  }
}
