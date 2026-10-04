resource "nextdns_profile" "home" {
  name = "Home"
}

# This resource manages the whole allowlist: domains added outside
# Terraform are removed on the next apply.
resource "nextdns_allowlist" "home" {
  profile_id = nextdns_profile.home.id

  domain {
    id = "example.com"
  }

  domain {
    id = "example.net"
  }

  domain {
    id = "example.org"
  }

  # Inactive entries stay in the list but have no effect.
  domain {
    id     = "example.dev"
    active = false
  }
}

# For long lists, a dynamic block creates one domain block per item.
locals {
  allowlist_domains = ["cdn.example.com", "login.example.com", "api.example.com"]
}

resource "nextdns_profile" "office" {
  name = "Office"
}

resource "nextdns_allowlist" "office" {
  profile_id = nextdns_profile.office.id

  dynamic "domain" {
    for_each = local.allowlist_domains
    content {
      id = domain.value
    }
  }
}
