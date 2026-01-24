package discord

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceDiscordServerOnboarding(t *testing.T) {
	testServerID := os.Getenv("DISCORD_TEST_SERVER_ID")
	if testServerID == "" {
		t.Skip("DISCORD_TEST_SERVER_ID envvar must be set for acceptance tests")
	}

	name := "discord_server_onboarding.example"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceDiscordServerOnboardingBasic(testServerID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "server_id", testServerID),
					resource.TestCheckResourceAttr(name, "enabled", "true"),
					resource.TestCheckResourceAttr(name, "mode", "0"),
					resource.TestCheckResourceAttr(name, "default_channel_ids.#", "1"),
				),
			},
			{
				Config: testAccResourceDiscordServerOnboardingWithPrompts(testServerID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "server_id", testServerID),
					resource.TestCheckResourceAttr(name, "enabled", "true"),
					resource.TestCheckResourceAttr(name, "mode", "1"),
					resource.TestCheckResourceAttr(name, "prompt.#", "1"),
					resource.TestCheckResourceAttr(name, "prompt.0.title", "Select options"),
					resource.TestCheckResourceAttr(name, "prompt.0.type", "0"),
					resource.TestCheckResourceAttr(name, "prompt.0.single_select", "false"),
					resource.TestCheckResourceAttr(name, "prompt.0.required", "true"),
					resource.TestCheckResourceAttr(name, "prompt.0.in_onboarding", "true"),
					resource.TestCheckResourceAttr(name, "prompt.0.option.#", "2"),
					resource.TestCheckResourceAttr(name, "prompt.0.option.0.title", "Option 1"),
					resource.TestCheckResourceAttr(name, "prompt.0.option.1.title", "Option 2"),
				),
			},
		},
	})
}

func testAccResourceDiscordServerOnboardingBasic(serverID string) string {
	return fmt.Sprintf(`
# Discord requires minimum 1 default channel for onboarding
resource "discord_text_channel" "ch_1" {
  server_id = "%[1]s"
  name      = "tf-test-channel-1"
}

resource "discord_server_onboarding" "example" {
  server_id = "%[1]s"
  enabled   = true
  mode      = 0

  default_channel_ids = [
    discord_text_channel.ch_1.id,
  ]
}`, serverID)
}

func testAccResourceDiscordServerOnboardingWithPrompts(serverID string) string {
	return fmt.Sprintf(`
resource "discord_text_channel" "ch_1" {
  server_id = "%[1]s"
  name      = "tf-test-channel-1"
}

resource "discord_text_channel" "ch_2" {
  server_id = "%[1]s"
  name      = "tf-test-channel-2"
}

resource "discord_text_channel" "ch_3" {
  server_id = "%[1]s"
  name      = "tf-test-channel-3"
}

resource "discord_role" "role_1" {
  server_id = "%[1]s"
  name      = "tf-test-role-1"
}

resource "discord_server_onboarding" "example" {
  server_id = "%[1]s"
  enabled   = true
  mode      = 1

  default_channel_ids = [
    discord_text_channel.ch_1.id,
  ]

  prompt {
    title         = "Select options"
    type          = 0
    single_select = false
    required      = true
    in_onboarding = true

    option {
      title       = "Option 1"
      description = "Test option 1"
      channel_ids = [discord_text_channel.ch_2.id]
      role_ids    = [discord_role.role_1.id]
    }

    option {
      title       = "Option 2"
      description = "Test option 2"
      channel_ids = [discord_text_channel.ch_3.id]
    }
  }
}`, serverID)
}
