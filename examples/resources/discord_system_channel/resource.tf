resource "discord_text_channel" "system" {
  name      = "discord-notifications"
  server_id = var.server_id
}

resource "discord_system_channel" "system" {
  server_id            = discord_text_channel.system.server_id
  system_channel_id    = discord_text_channel.system.id
  system_channel_flags = 6 # Suppress premium subscriptions (2) + server setup tips (4)
}
