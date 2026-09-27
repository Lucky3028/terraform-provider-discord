resource "discord_category_channel" "general" {
  name      = "General"
  server_id = var.server_id
}

resource "discord_text_channel" "announcements" {
  name      = "announcements"
  server_id = var.server_id
  category  = discord_category_channel.general.id
}

resource "discord_text_channel" "general_chat" {
  name      = "general-chat"
  server_id = var.server_id
  category  = discord_category_channel.general.id
}

resource "discord_text_channel" "off_topic" {
  name      = "off-topic"
  server_id = var.server_id
  category  = discord_category_channel.general.id
}

# Order channels within a category atomically via Discord's bulk reorder
# endpoint. Index 0 is the topmost channel inside the category.
resource "discord_channel_order" "general" {
  server_id   = var.server_id
  category_id = discord_category_channel.general.id

  channel_ids = [
    discord_text_channel.announcements.id,
    discord_text_channel.general_chat.id,
    discord_text_channel.off_topic.id,
  ]
}

# Order top-level channels and categories (those with no parent).
resource "discord_channel_order" "top_level" {
  server_id = var.server_id

  channel_ids = [
    discord_category_channel.general.id,
  ]
}
