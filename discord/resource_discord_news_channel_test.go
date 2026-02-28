package discord

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceDiscordNewsChannel(t *testing.T) {
	testServerID := os.Getenv("DISCORD_TEST_SERVER_ID")
	if testServerID == "" {
		t.Skip("DISCORD_TEST_SERVER_ID envvar must be set for acceptance tests")
	}
	name := "discord_news_channel.example"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceDiscordNewsChannel(testServerID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "server_id", testServerID),
					resource.TestCheckResourceAttr(name, "name", "terraform-news-channel"),
					resource.TestCheckResourceAttr(name, "type", "news"),
					resource.TestCheckResourceAttr(name, "position", "1"),
					resource.TestCheckResourceAttrSet(name, "channel_id"),
					resource.TestCheckResourceAttr(name, "topic", "Testing news channel"),
					resource.TestCheckResourceAttr(name, "nsfw", "false"),
					resource.TestCheckResourceAttr(name, "sync_perms_with_category", "false"),
				),
			},
		},
	})
}

func testAccResourceDiscordNewsChannel(serverID string) string {
	return fmt.Sprintf(`
	resource "discord_news_channel" "example" {
	  server_id = "%[1]s"
      name = "terraform-news-channel"
      position = 1
      topic = "Testing news channel"
      nsfw = false
      sync_perms_with_category = false
	}`, serverID)
}
