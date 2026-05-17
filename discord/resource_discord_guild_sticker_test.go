package discord

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceDiscordGuildSticker(t *testing.T) {
	testResourceDiscordGuildSticker := `
		resource "discord_guild_sticker" "test" {
			server_id   = var.server_id
			name        = "test-sticker"
			description = "A test sticker"
			tags        = "test"
			file        = "testdata/sticker.png"
		}
	`

	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testResourceDiscordGuildSticker,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("discord_guild_sticker.test", "name", "test-sticker"),
					resource.TestCheckResourceAttr("discord_guild_sticker.test", "description", "A test sticker"),
					resource.TestCheckResourceAttr("discord_guild_sticker.test", "tags", "test"),
				),
			},
		},
	})
}
