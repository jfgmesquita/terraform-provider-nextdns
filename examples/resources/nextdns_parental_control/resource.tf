resource "nextdns_profile" "office" {
  name = "Office"
}

resource "nextdns_parental_control" "office" {
  profile_id = nextdns_profile.office.id

  # Keys are IDs from https://api.nextdns.io/parentalcontrol/services
  services = {
    tiktok  = {}
    steam   = {}
    netflix = { recreation = true } # allowed during the lunch break
    twitch  = { active = false }
  }

  # Keys are IDs from https://api.nextdns.io/parentalcontrol/categories
  categories = {
    gambling          = {}
    gaming            = {}
    "social-networks" = { recreation = true }
  }

  # Lunch break on weekdays.
  recreation {
    timezone  = "Europe/Lisbon"
    monday    = { start = "12:30", end = "14:00" }
    tuesday   = { start = "12:30", end = "14:00" }
    wednesday = { start = "12:30", end = "14:00" }
    thursday  = { start = "12:30", end = "14:00" }
    friday    = { start = "12:30", end = "14:00" }
  }

  safe_search             = true
  youtube_restricted_mode = false
  block_bypass            = false
}
