# Built-in profanity / sexual content / slurs filter, alerting a mod log channel.
resource "discord_auto_moderation_rule" "offensive_language" {
  server_id    = var.server_id
  name         = "Offensive language"
  event_type   = 1 # MESSAGE_SEND
  trigger_type = 4 # KEYWORD_PRESET

  trigger_metadata {
    presets = [1, 2, 3] # profanity, sexual_content, slurs
  }

  exempt_roles = var.moderator_role_ids

  actions {
    type = 1 # BLOCK_MESSAGE
    metadata {
      custom_message = "Message blocked. Offensive language is not allowed."
    }
  }
  actions {
    type = 2 # SEND_ALERT_MESSAGE
    metadata {
      channel_id = var.mod_log_channel_id
    }
  }
}

# Mention-spam protection with a 5-mention cap.
resource "discord_auto_moderation_rule" "mention_spam" {
  server_id    = var.server_id
  name         = "Anti mention spam"
  event_type   = 1
  trigger_type = 5 # MENTION_SPAM

  trigger_metadata {
    mention_total_limit             = 5
    mention_raid_protection_enabled = true
  }

  exempt_roles = var.moderator_role_ids

  actions {
    type = 1
    metadata {
      custom_message = "Too many mentions in a single message."
    }
  }
  actions {
    type = 2
    metadata {
      channel_id = var.mod_log_channel_id
    }
  }
}

# Regex link filter — blocks any http(s) link.
resource "discord_auto_moderation_rule" "link_filter" {
  server_id    = var.server_id
  name         = "Link filter"
  event_type   = 1
  trigger_type = 1 # KEYWORD

  trigger_metadata {
    regex_patterns = ["https?://"]
  }

  exempt_roles = var.moderator_role_ids

  actions {
    type = 1
    metadata {
      custom_message = "Links are not allowed. Contact a moderator."
    }
  }
  actions {
    type = 2
    metadata {
      channel_id = var.mod_log_channel_id
    }
  }
}
