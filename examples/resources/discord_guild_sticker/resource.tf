resource "discord_guild_sticker" "example" {
  server_id   = var.server_id
  name        = "wave"
  description = "Waving hello"
  tags        = "👋"
  file        = "${path.module}/assets/wave.png"
}
