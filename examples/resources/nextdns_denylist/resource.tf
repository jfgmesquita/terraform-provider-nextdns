resource "nextdns_profile" "home" {
  name = "Home"
}

# This resource manages the whole denylist: domains added outside
# Terraform are removed on the next apply.
resource "nextdns_denylist" "home" {
  profile_id = nextdns_profile.home.id

  domain {
    id = "ads.example.com"
  }

  domain {
    id = "tracker.example.com"
  }

  domain {
    id = "telemetry.example.com"
  }

  # Inactive entries stay in the list but have no effect.
  domain {
    id     = "analytics.example.com"
    active = false
  }
}

# For long lists, a dynamic block creates one domain block per item.
locals {
  denylist_domains = ["ads.example.net", "ads.example.org", "tracking.example.net"]
}

resource "nextdns_profile" "office" {
  name = "Office"
}

resource "nextdns_denylist" "office" {
  profile_id = nextdns_profile.office.id

  dynamic "domain" {
    for_each = local.denylist_domains
    content {
      id = domain.value
    }
  }
}
