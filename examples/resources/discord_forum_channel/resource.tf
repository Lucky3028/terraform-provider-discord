resource "discord_forum_channel" "announcements" {
  name      = "announcements"
  server_id = var.server_id
  position  = 0
  topic     = "Important announcements. React with ✅ when you have read a post."

  available_tags {
    name       = "Information"
    emoji_name = "📢"
  }
  available_tags {
    name       = "Important"
    emoji_name = "⚠️"
    moderated  = true # only moderators can apply this tag
  }
  available_tags {
    name       = "Question"
    emoji_name = "❓"
  }

  default_reaction_emoji {
    emoji_name = "✅"
  }

  default_sort_order   = 0 # latest activity first
  default_forum_layout = 1 # list view
}
