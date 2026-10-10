resource "nextdns_profile" "office" {
  name = "Office"
}

# For long lists, a dynamic block creates one domain block per item.
locals {
  denylist_domains = ["example.com", "example.org", "example.net"]
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
