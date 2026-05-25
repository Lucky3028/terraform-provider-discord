package discord

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// rateLimitPerUserSchema returns the slowmode schema used by text, news, and
// forum channels. Discord caps the value at 21600 seconds (6 hours).
func rateLimitPerUserSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeInt,
		Optional:    true,
		Default:     0,
		Description: "Slowmode: minimum number of seconds a user has to wait between sending messages. `0` disables slowmode. Discord caps the value at `21600` (6 hours).",
		ValidateFunc: func(val interface{}, key string) (warns []string, errors []error) {
			v := val.(int)
			if v < 0 || v > 21600 {
				errors = append(errors, fmt.Errorf("%s must be between 0 and 21600 seconds, got: %d", key, v))
			}
			return
		},
	}
}

func getTextChannelType(channelType discordgo.ChannelType) (string, bool) {
	switch channelType {
	case 0:
		return "text", true
	case 2:
		return "voice", true
	case 4:
		return "category", true
	case 5:
		return "news", true
	case 6:
		return "store", true
	case 15:
		return "forum", true
	}

	return "text", false
}

func getDiscordChannelType(name string) (discordgo.ChannelType, bool) {
	switch name {
	case "text":
		return discordgo.ChannelTypeGuildText, true
	case "voice":
		return discordgo.ChannelTypeGuildVoice, true
	case "category":
		return discordgo.ChannelTypeGuildCategory, true
	case "news":
		return discordgo.ChannelTypeGuildNews, true
	case "store":
		return discordgo.ChannelTypeGuildStore, true
	case "forum":
		return discordgo.ChannelTypeGuildForum, true
	}

	return 0, false
}

type Channel struct {
	ServerId  string
	ChannelId string
	Channel   *discordgo.Channel
}

func findChannelById(array []*discordgo.Channel, id string) *discordgo.Channel {
	for _, element := range array {
		if element.ID == id {
			return element
		}
	}

	return nil
}

func arePermissionsSynced(from *discordgo.Channel, to *discordgo.Channel) bool {
	for _, p1 := range from.PermissionOverwrites {
		cont := false
		for _, p2 := range to.PermissionOverwrites {
			if p1.ID == p2.ID && p1.Type == p2.Type && p1.Allow == p2.Allow && p1.Deny == p2.Deny {
				cont = true
				break
			}
		}
		if !cont {
			return false
		}
	}

	for _, p1 := range to.PermissionOverwrites {
		cont := false
		for _, p2 := range from.PermissionOverwrites {
			if p1.ID == p2.ID && p1.Type == p2.Type && p1.Allow == p2.Allow && p1.Deny == p2.Deny {
				cont = true
				break
			}
		}
		if !cont {
			return false
		}
	}

	return true
}

func syncChannelPermissions(c *discordgo.Session, ctx context.Context, from *discordgo.Channel, to *discordgo.Channel) error {
	for _, p := range to.PermissionOverwrites {
		if err := c.ChannelPermissionDelete(to.ID, p.ID); err != nil {
			return err
		}
	}

	for _, p := range from.PermissionOverwrites {
		if err := c.ChannelPermissionSet(to.ID, p.ID, discordgo.PermissionOverwriteTypeRole, p.Allow, p.Deny, discordgo.WithContext(ctx)); err != nil {
			return err
		}
	}

	return nil
}

func getDiscordChannelPermissionType(value string) (discordgo.PermissionOverwriteType, bool) {
	switch value {
	case "role":
		return discordgo.PermissionOverwriteTypeRole, true
	case "user":
		return discordgo.PermissionOverwriteTypeMember, true
	default:
		return 0, false
	}
}
