resource "discord_server_onboarding" "example" {
  server_id = var.server_id
  enabled   = true
  mode      = 1 # Advanced mode (0 = Default, 1 = Advanced)

  # Minimum 1 channel required by Discord
  default_channel_ids = [
    discord_text_channel.general.id,
  ]

  prompt {
    title         = "What are your interests?"
    type          = 0 # Multiple choice
    single_select = false
    required      = true
    in_onboarding = true

    option {
      title       = "Gaming"
      description = "Access gaming channels and get the gamer role"
      emoji_name  = "🎮"
      channel_ids = [discord_text_channel.gaming.id]
      role_ids    = [discord_role.gamer.id]
    }

    option {
      title       = "Development"
      description = "Join development discussion channels"
      emoji_name  = "💻"
      channel_ids = [
        discord_text_channel.dev_chat.id,
        discord_text_channel.code_help.id
      ]
      role_ids = [discord_role.developer.id]
    }

    option {
      title       = "Community"
      description = "Join general community channels"
      emoji_name  = "❤️"
      channel_ids = [discord_text_channel.community.id]
    }
  }

  prompt {
    title         = "What is your experience level?"
    type          = 1 # Dropdown
    single_select = true
    required      = false
    in_onboarding = true

    option {
      title       = "Beginner"
      description = "New to the community"
      role_ids    = [discord_role.beginner.id]
    }

    option {
      title       = "Intermediate"
      description = "Some experience"
      role_ids    = [discord_role.intermediate.id]
    }

    option {
      title       = "Expert"
      description = "Experienced member"
      role_ids    = [discord_role.expert.id]
    }
  }
}
