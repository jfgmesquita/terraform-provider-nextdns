resource "nextdns_profile" "office" {
  name = "Office"
}

resource "nextdns_rewrites" "office" {
  profile_id = nextdns_profile.office.id

  # IPv4 address: NextDNS answers with an A record.
  rewrite {
    domain = "example.com"
    answer = "192.168.1.1"
  }

  # IPv6 address: NextDNS answers with an AAAA record.
  rewrite {
    domain = "example.org"
    answer = "fd00::1"
  }

  # Domain name: NextDNS answers with a CNAME record.
  rewrite {
    domain = "example.net"
    answer = "example.com"
  }
}
