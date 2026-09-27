package discord

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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

// The tests below cover what the acceptance test above cannot: it is skipped
// without DISCORD_TEST_SERVER_ID, so it never ran in CI and never caught that
// topic and nsfw were absent from the forum branch of the create and update
// payloads. Discord's API silently accepted the result — topic is sent
// `omitempty`, so an empty one is dropped rather than rejected — which left the
// forum's post guidelines unset and the plan permanently non-empty.

func TestBuildChannelCreateDataForumSendsTopicAndNSFW(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDiscordForumChannel().Schema, map[string]interface{}{
		"server_id":           "1234",
		"type":                "forum",
		"name":                "terraform-forum-channel",
		"topic":               "Testing forum channel",
		"nsfw":                true,
		"rate_limit_per_user": 10,
	})

	data := buildChannelCreateData(d, discordgo.ChannelTypeGuildForum)

	if data.Topic != "Testing forum channel" {
		t.Errorf("topic: expected %q, got %q", "Testing forum channel", data.Topic)
	}
	if !data.NSFW {
		t.Error("nsfw: expected true, got false")
	}
	if data.RateLimitPerUser != 10 {
		t.Errorf("rate_limit_per_user: expected 10, got %d", data.RateLimitPerUser)
	}
	if data.Type != discordgo.ChannelTypeGuildForum {
		t.Errorf("type: expected %d, got %d", discordgo.ChannelTypeGuildForum, data.Type)
	}
}

func TestBuildChannelEditForumSendsChangedTopic(t *testing.T) {
	d := forumResourceDataWithDiff(t,
		map[string]string{
			"server_id":           "1234",
			"type":                "forum",
			"name":                "terraform-forum-channel",
			"topic":               "Old guidelines",
			"rate_limit_per_user": "10",
		},
		map[string]interface{}{
			"server_id":           "1234",
			"type":                "forum",
			"name":                "terraform-forum-channel",
			"topic":               "New guidelines",
			"rate_limit_per_user": 10,
		},
	)

	// Guards the test itself: without a real diff HasChange is false throughout
	// and the assertion below would pass for the wrong reason.
	if !d.HasChange("topic") {
		t.Fatal("expected the diff to report topic as changed")
	}

	edit := buildChannelEdit(d, &discordgo.Channel{
		Name:             "terraform-forum-channel",
		Topic:            "Old guidelines",
		RateLimitPerUser: 10,
	})

	if edit.Topic != "New guidelines" {
		t.Errorf("topic: expected %q, got %q", "New guidelines", edit.Topic)
	}
}

func TestBuildChannelEditForumKeepsUnchangedTopic(t *testing.T) {
	attrs := map[string]string{
		"server_id":           "1234",
		"type":                "forum",
		"name":                "terraform-forum-channel",
		"topic":               "Guidelines",
		"rate_limit_per_user": "10",
	}
	d := forumResourceDataWithDiff(t, attrs, map[string]interface{}{
		"server_id":           "1234",
		"type":                "forum",
		"name":                "renamed-forum-channel",
		"topic":               "Guidelines",
		"rate_limit_per_user": 10,
	})

	edit := buildChannelEdit(d, &discordgo.Channel{
		Name:             "terraform-forum-channel",
		Topic:            "Guidelines",
		RateLimitPerUser: 10,
	})

	if edit.Name != "renamed-forum-channel" {
		t.Errorf("name: expected %q, got %q", "renamed-forum-channel", edit.Name)
	}
	// A rename must carry the existing topic along. Discord drops an empty topic
	// instead of clearing it, so a regression here is invisible until the next
	// plan, which then never converges.
	if edit.Topic != "Guidelines" {
		t.Errorf("topic: expected %q to survive a rename, got %q", "Guidelines", edit.Topic)
	}
}

// forumResourceDataWithDiff builds a ResourceData backed by a real plan diff, so
// HasChange answers the way it does during an apply. TestResourceDataRaw cannot:
// it has no prior state to diff the config against, leaving HasChange false for
// every attribute.
func forumResourceDataWithDiff(t *testing.T, state map[string]string, config map[string]interface{}) *schema.ResourceData {
	t.Helper()

	res := resourceDiscordForumChannel()
	is := &terraform.InstanceState{ID: "1482795207656214762", Attributes: state}

	diff, err := res.Diff(context.Background(), is, terraform.NewResourceConfigRaw(config), nil)
	if err != nil {
		t.Fatalf("diff: %s", err)
	}

	d, err := schema.InternalMap(res.Schema).Data(is, diff)
	if err != nil {
		t.Fatalf("data: %s", err)
	}

	return d
}
