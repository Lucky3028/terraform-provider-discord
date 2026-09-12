resource "discord_text_channel" "rules" {
  name      = "rules"
  server_id = var.server_id
}

resource "discord_rules_channel" "rules" {
  server_id        = var.server_id
  rules_channel_id = discord_text_channel.rules.id
}
