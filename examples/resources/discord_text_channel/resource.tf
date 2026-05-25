resource "discord_text_channel" "general" {
  name      = "general"
  server_id = var.server_id
  position  = 0
}

resource "discord_text_channel" "announcements" {
  name                = "announcements"
  server_id           = var.server_id
  position            = 1
  topic               = "Important announcements only."
  rate_limit_per_user = 60 # 60-second slowmode
}
