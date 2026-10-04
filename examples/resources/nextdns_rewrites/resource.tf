resource "nextdns_profile" "home" {
  name = "Home"
}

# This resource manages all rewrites of the profile: rewrites added
# outside Terraform are removed on the next apply.
resource "nextdns_rewrites" "home" {
  profile_id = nextdns_profile.home.id

  # IPv4 address: NextDNS answers with an A record.
  rewrite {
    domain = "router.home"
    answer = "192.168.1.1"
  }

  # IPv6 address: NextDNS answers with an AAAA record.
  rewrite {
    domain = "router.home"
    answer = "fd00::1"
  }

  # Domain name: NextDNS answers with a CNAME record.
  rewrite {
    domain = "www.example.com"
    answer = "example.org"
  }
}

# For long lists, a dynamic block creates one rewrite block per item.
locals {
  hosts = {
    "nas.home"     = "192.168.1.10"
    "printer.home" = "192.168.1.20"
  }
}

resource "nextdns_profile" "office" {
  name = "Office"
}

resource "nextdns_rewrites" "office" {
  profile_id = nextdns_profile.office.id

  dynamic "rewrite" {
    for_each = local.hosts
    content {
      domain = rewrite.key
      answer = rewrite.value
    }
  }
}
