resource "discord_voice_channel" "lobby" {
  name      = "lobby"
  server_id = var.server_id
}

resource "discord_server_widget" "example" {
  server_id  = var.server_id
  enabled    = true
  channel_id = discord_voice_channel.lobby.id
}
