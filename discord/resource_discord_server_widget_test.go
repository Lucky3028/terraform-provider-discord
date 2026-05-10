package discord

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceDiscordServerWidget(t *testing.T) {
	testServerID := os.Getenv("DISCORD_TEST_SERVER_ID")
	if testServerID == "" {
		t.Skip("DISCORD_TEST_SERVER_ID envvar must be set for acceptance tests")
	}

	name := "discord_server_widget.example"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceDiscordServerWidget(testServerID, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "server_id", testServerID),
					resource.TestCheckResourceAttr(name, "enabled", "true"),
					resource.TestCheckResourceAttrSet(name, "channel_id"),
				),
			},
			{
				Config: testAccResourceDiscordServerWidget(testServerID, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "enabled", "false"),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccResourceDiscordServerWidget(serverID string, enabled bool) string {
	return fmt.Sprintf(`
	resource "discord_voice_channel" "test_widget" {
	  server_id                = "%[1]s"
	  name                     = "terraform-test-widget"
	  sync_perms_with_category = false
	}

	resource "discord_server_widget" "example" {
	  server_id  = "%[1]s"
	  enabled    = %[2]t
	  channel_id = discord_voice_channel.test_widget.id
	}`, serverID, enabled)
}
