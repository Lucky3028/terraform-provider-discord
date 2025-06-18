package discord

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceDiscordForumChannel(t *testing.T) {
	testServerID := os.Getenv("DISCORD_TEST_SERVER_ID")
	if testServerID == "" {
		t.Skip("DISCORD_TEST_SERVER_ID envvar must be set for acceptance tests")
	}
	name := "discord_forum_channel.example"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceDiscordForumChannel(testServerID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "server_id", testServerID),
					resource.TestCheckResourceAttr(name, "name", "terraform-forum-channel"),
					resource.TestCheckResourceAttr(name, "type", "forum"),
					resource.TestCheckResourceAttr(name, "position", "1"),
					resource.TestCheckResourceAttrSet(name, "channel_id"),
					resource.TestCheckResourceAttr(name, "topic", "Testing forum channel"),
					resource.TestCheckResourceAttr(name, "nsfw", "false"),
					resource.TestCheckResourceAttr(name, "sync_perms_with_category", "false"),
				),
			},
		},
	})
}

func testAccResourceDiscordForumChannel(serverID string) string {
	return fmt.Sprintf(`
	resource "discord_forum_channel" "example" {
	  server_id = "%[1]s"
      name = "terraform-forum-channel"
      position = 1
      topic = "Testing forum channel"
      nsfw = false
      sync_perms_with_category = false
	}`, serverID)
}
