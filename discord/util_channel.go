package discord

import (
	"context"
	"sort"

	"github.com/bwmarrin/discordgo"
)

// siblingPosition returns the index of `channel` among its same-parent
// siblings when sorted by Discord's stored position. Discord's
// `Channel.position` is a guild-wide flat counter, while HCL typically
// expresses positions relative to siblings within a category (0, 1, 2, ...).
// Returning the sibling index keeps Terraform state in sync with how users
// write channel positions.
//
// Categories themselves are sorted guild-wide and should not be passed here.
func siblingPosition(ctx context.Context, client *discordgo.Session, channel *discordgo.Channel) (int, error) {
	all, err := client.GuildChannels(channel.GuildID, discordgo.WithContext(ctx))
	if err != nil {
		return 0, err
	}
	siblings := make([]*discordgo.Channel, 0)
	for _, c := range all {
		if c.Type == discordgo.ChannelTypeGuildCategory {
			continue
		}
		if c.ParentID != channel.ParentID {
			continue
		}
		siblings = append(siblings, c)
	}
	sort.SliceStable(siblings, func(i, j int) bool {
		if siblings[i].Position != siblings[j].Position {
			return siblings[i].Position < siblings[j].Position
		}
		// Tie-break on ID for stable ordering when Discord stores equal positions.
		return siblings[i].ID < siblings[j].ID
	})
	for i, c := range siblings {
		if c.ID == channel.ID {
			return i, nil
		}
	}
	// Channel wasn't found among its declared siblings — fall back to the
	// guild-wide value to avoid masking the inconsistency.
	return channel.Position, nil
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
