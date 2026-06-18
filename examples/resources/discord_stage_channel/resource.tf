resource "discord_stage_channel" "events" {
  server_id = var.server_id
  name      = "Events Stage"
}
