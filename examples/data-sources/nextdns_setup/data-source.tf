resource "nextdns_profile" "office" {
  name = "Office"
}

data "nextdns_setup" "office" {
  profile_id = nextdns_profile.office.id
}

output "office_doh_url" {
  value = data.nextdns_setup.office.doh_url
}

output "office_dns_servers" {
  value = concat(data.nextdns_setup.office.ipv6, data.nextdns_setup.office.linked_ip_servers)
}

output "office_update_token" {
  value     = data.nextdns_setup.office.linked_ip_update_token
  sensitive = true
}
