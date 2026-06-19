package discord

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Note: the test server (DISCORD_TEST_SERVER_ID) must have the Community
// feature enabled, otherwise Discord rejects rules_channel_id.
func TestAccResourceDiscordRulesChannel(t *testing.T) {
	testServerID := os.Getenv("DISCORD_TEST_SERVER_ID")
	if testServerID == "" {
		t.Skip("DISCORD_TEST_SERVER_ID envvar must be set for acceptance tests")
	}

	name := "discord_rules_channel.example"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceDiscordRulesChannel(testServerID, "discord_text_channel.test_rules_a"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "server_id", testServerID),
					resource.TestCheckResourceAttrSet(name, "rules_channel_id"),
				),
			},
			{
				Config: testAccResourceDiscordRulesChannel(testServerID, "discord_text_channel.test_rules_b"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(name, "rules_channel_id"),
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

func testAccResourceDiscordRulesChannel(serverID, channelRef string) string {
	return fmt.Sprintf(`
	resource "discord_text_channel" "test_rules_a" {
	  server_id                = "%[1]s"
	  name                     = "terraform-test-rules-a"
	  sync_perms_with_category = false
	}

	resource "discord_text_channel" "test_rules_b" {
	  server_id                = "%[1]s"
	  name                     = "terraform-test-rules-b"
	  sync_perms_with_category = false
	}

	resource "discord_rules_channel" "example" {
	  server_id        = "%[1]s"
	  rules_channel_id = %[2]s.id
	}`, serverID, channelRef)
}
