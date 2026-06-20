resource "discord_member_nickname" "jake" {
  user_id   = var.user_id
  server_id = var.server_id
  nick      = "Jake"
}
