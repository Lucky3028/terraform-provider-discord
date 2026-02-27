package discord

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceDiscordSystemChannel(t *testing.T) {
	testServerID := os.Getenv("DISCORD_TEST_SERVER_ID")
	if testServerID == "" {
		t.Skip("DISCORD_TEST_SERVER_ID envvar must be set for acceptance tests")
	}

	name := "discord_system_channel.example"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceDiscordSystemChannelFlags(testServerID, 1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "server_id", testServerID),
					resource.TestCheckResourceAttrSet(name, "system_channel_id"),
					resource.TestCheckResourceAttr(name, "system_channel_flags", "1"),
				),
			},
			{
				Config: testAccResourceDiscordSystemChannelFlags(testServerID, 3),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "system_channel_flags", "3"),
				),
			},
		},
	})
}

func testAccResourceDiscordSystemChannelFlags(serverID string, flags int) string {
	return fmt.Sprintf(`
	resource "discord_text_channel" "test_system" {
	  server_id                = "%[1]s"
	  name                     = "terraform-test-system"
	  sync_perms_with_category = false
	}

	resource "discord_system_channel" "example" {
	  server_id            = "%[1]s"
	  system_channel_id    = discord_text_channel.test_system.id
	  system_channel_flags = %[2]d
	}`, serverID, flags)
}
