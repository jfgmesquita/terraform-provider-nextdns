resource "nextdns_profile" "office" {
  name = "Office"
}

# Every setting is listed here. Any setting left out is turned off, and any
# service, category or recreation time left out is removed.
resource "nextdns_parental_control" "office" {
  profile_id = nextdns_profile.office.id

  safe_search             = true
  youtube_restricted_mode = false
  block_bypass            = true # VPNs, proxies and other DNS providers

  # Keys are IDs from https://api.nextdns.io/parentalcontrol/services
  services = {
    tiktok  = {}
    steam   = {}
    netflix = { recreation = true } # allowed during the lunch break
    twitch  = { active = false }    # stays in the list, but has no effect
  }

  # Keys are IDs from https://api.nextdns.io/parentalcontrol/categories
  categories = {
    gambling          = {}
    gaming            = {}
    "social-networks" = { recreation = true }
  }

  # Lunch break on weekdays. Days left out have no recreation time.
  recreation {
    timezone  = "Europe/Lisbon"
    monday    = { start = "12:30", end = "14:00" }
    tuesday   = { start = "12:30", end = "14:00" }
    wednesday = { start = "12:30", end = "14:00" }
    thursday  = { start = "12:30", end = "14:00" }
    friday    = { start = "12:30", end = "14:00" }
  }
}
