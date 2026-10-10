resource "nextdns_profile" "office" {
  name = "Office"
}

# For long lists, a dynamic block creates one rewrite block per item.
locals {
  rewrites = {
    "example.com" = "192.168.1.10"
    "example.org" = "192.168.1.20"
    "example.net" = "192.168.1.30"
  }
}

resource "nextdns_rewrites" "office" {
  profile_id = nextdns_profile.office.id

  dynamic "rewrite" {
    for_each = local.rewrites
    content {
      domain = rewrite.key
      answer = rewrite.value
    }
  }
}
