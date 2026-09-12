package discord

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceDiscordStageChannel(t *testing.T) {
	testServerID := os.Getenv("DISCORD_TEST_SERVER_ID")
	if testServerID == "" {
		t.Skip("DISCORD_TEST_SERVER_ID envvar must be set for acceptance tests")
	}
	name := "discord_stage_channel.example"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceDiscordStageChannel(testServerID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "server_id", testServerID),
					resource.TestCheckResourceAttr(name, "name", "terraform-stage-channel"),
					resource.TestCheckResourceAttr(name, "type", "stage"),
					resource.TestCheckResourceAttr(name, "position", "1"),
					resource.TestCheckResourceAttr(name, "bitrate", "64000"),
					resource.TestCheckResourceAttr(name, "user_limit", "10000"),
					resource.TestCheckResourceAttr(name, "sync_perms_with_category", "false"),
					resource.TestCheckResourceAttrSet(name, "channel_id"),
				),
			},
		},
	})
}

func testAccResourceDiscordStageChannel(serverID string) string {
	return fmt.Sprintf(`
	resource "discord_stage_channel" "example" {
	  server_id = "%[1]s"
      name = "terraform-stage-channel"
      position = 1
      bitrate = 64000
      user_limit = 10000
      sync_perms_with_category = false
	}`, serverID)
}
