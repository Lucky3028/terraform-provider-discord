resource "discord_forum_channel" "announcements" {
  name      = "announcements"
  server_id = var.server_id

  available_tags {
    name       = "Information"
    emoji_name = "📢"
  }
}

resource "discord_thread" "welcome" {
  channel_id = discord_forum_channel.announcements.id
  name       = "Welcome to the announcements forum"

  message {
    content = <<-EOT
      Welcome! Here's how to use this forum:

      - Open a new post for each announcement
      - Reply in the thread to discuss
      - Use the `Information` tag for general updates
    EOT
  }

  applied_tags = [
    # Pick the Information tag's computed id (tags are ordered as declared).
    discord_forum_channel.announcements.available_tags[0].id,
  ]

  auto_archive_duration = 4320 # 3 days
}
