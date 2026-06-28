package discord

import (
	"fmt"
	"os"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestFindMemberByUsername(t *testing.T) {
	legacy := &discordgo.Member{User: &discordgo.User{Username: "alice", Discriminator: "1234"}}
	migrated := &discordgo.Member{User: &discordgo.User{Username: "bob", Discriminator: "0"}}
	members := []*discordgo.Member{legacy, migrated}

	tests := []struct {
		name          string
		username      string
		discriminator string
		want          *discordgo.Member
	}{
		{"migrated account, discriminator omitted", "bob", "", migrated},
		{"migrated account, discriminator \"0\"", "bob", "0", migrated},
		{"legacy account, matching discriminator", "alice", "1234", legacy},
		{"legacy account, discriminator omitted", "alice", "", legacy},
		{"legacy account, wrong discriminator", "alice", "9999", nil},
		{"unknown username", "carol", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findMemberByUsername(members, tt.username, tt.discriminator); got != tt.want {
				t.Errorf("findMemberByUsername(%q, %q) = %v, want %v", tt.username, tt.discriminator, got, tt.want)
			}
		})
	}
}

func TestAccDatasourceDiscordMember(t *testing.T) {
	testServerID := os.Getenv("DISCORD_TEST_SERVER_ID")
	testUserID := os.Getenv("DISCORD_TEST_USER_ID")
	testUsername := os.Getenv("DISCORD_TEST_USERNAME")
	if testServerID == "" || testUserID == "" || testUsername == "" {
		t.Skip("DISCORD_TEST_SERVER_ID, DISCORD_TEST_USER_ID, and DISCORD_TEST_USERNAME envvars must be set for acceptance tests")
	}

	name := "data.discord_member.example"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatasourceDiscordMemberUserID(testServerID, testUserID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "server_id", testServerID),
					resource.TestCheckResourceAttr(name, "user_id", testUserID),
					resource.TestCheckResourceAttrSet(name, "joined_at"),
					resource.TestCheckResourceAttrSet(name, "avatar"),
					resource.TestCheckResourceAttrSet(name, "roles.#"),
					resource.TestCheckResourceAttr(name, "in_server", "true"),
				),
			},
			{
				Config: testAccDatasourceDiscordMemberUsername(testServerID, testUsername),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "server_id", testServerID),
					resource.TestCheckResourceAttr(name, "username", testUsername),
					resource.TestCheckResourceAttrSet(name, "joined_at"),
					resource.TestCheckResourceAttrSet(name, "avatar"),
					resource.TestCheckResourceAttrSet(name, "roles.#"),
					resource.TestCheckResourceAttr(name, "in_server", "true"),
				),
			},
		},
	})
}

func testAccDatasourceDiscordMemberUserID(serverId string, userID string) string {
	return fmt.Sprintf(`
	data "discord_member" "example" {
	  server_id = "%[1]s"
      user_id = "%[2]s"
	}`, serverId, userID)
}

func testAccDatasourceDiscordMemberUsername(serverId string, username string) string {
	return fmt.Sprintf(`
	data "discord_member" "example" {
	  server_id = "%[1]s"
	  username = "%[2]s"
      discriminator = "0"
	}`, serverId, username)
}
