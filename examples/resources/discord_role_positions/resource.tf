resource "discord_role" "admin" {
  server_id = var.server_id
  name      = "Admin"
}

resource "discord_role" "moderator" {
  server_id = var.server_id
  name      = "Moderator"
}

resource "discord_role" "member" {
  server_id = var.server_id
  name      = "Member"
}

# Order roles atomically. Discord's per-role position updates race against
# each other when several roles change at once, so this single PATCH is the
# safest way to lay out a large hierarchy.
resource "discord_role_positions" "main" {
  server_id = var.server_id

  position {
    role_id  = discord_role.admin.id
    position = 10
  }

  position {
    role_id  = discord_role.moderator.id
    position = 9
  }

  position {
    role_id  = discord_role.member.id
    position = 1
  }
}
