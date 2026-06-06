package discord

import (
	"context"
	"encoding/json"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Forum-specific channel fields are managed via a separate PATCH after the
// basic channel Create/Update so we don't rely on discordgo's typed API
// covering every Discord feature added since the library's last release. The
// wire format below mirrors Discord's REST representation directly.
type forumChannelPatch struct {
	AvailableTags        *[]forumTag        `json:"available_tags,omitempty"`
	DefaultReactionEmoji *forumDefaultEmoji `json:"default_reaction_emoji,omitempty"`
	DefaultSortOrder     *int               `json:"default_sort_order,omitempty"`
	DefaultForumLayout   *int               `json:"default_forum_layout,omitempty"`
}

type forumTag struct {
	ID        string  `json:"id,omitempty"`
	Name      string  `json:"name"`
	Moderated bool    `json:"moderated"`
	EmojiID   *string `json:"emoji_id"`
	EmojiName *string `json:"emoji_name"`
}

type forumDefaultEmoji struct {
	EmojiID   *string `json:"emoji_id"`
	EmojiName *string `json:"emoji_name"`
}

// channelReadResponse is a slim subset of Discord's channel object — just the
// forum-specific fields. Used to fill state in Read since these aren't exposed
// by older discordgo Channel structs.
type channelReadResponse struct {
	AvailableTags        []forumTag         `json:"available_tags"`
	DefaultReactionEmoji *forumDefaultEmoji `json:"default_reaction_emoji"`
	DefaultSortOrder     *int               `json:"default_sort_order"`
	DefaultForumLayout   *int               `json:"default_forum_layout"`
}

func buildForumChannelPatch(d *schema.ResourceData, includeUnset bool) (forumChannelPatch, bool) {
	patch := forumChannelPatch{}
	hasAny := false

	if v, ok := d.GetOk("available_tags"); ok {
		tags := expandForumTags(v.([]interface{}))
		patch.AvailableTags = &tags
		hasAny = true
	} else if includeUnset && d.HasChange("available_tags") {
		empty := []forumTag{}
		patch.AvailableTags = &empty
		hasAny = true
	}

	if v, ok := d.GetOk("default_reaction_emoji"); ok {
		blocks := v.([]interface{})
		if len(blocks) > 0 {
			patch.DefaultReactionEmoji = expandForumDefaultEmoji(blocks[0].(map[string]interface{}))
			hasAny = true
		}
	} else if includeUnset && d.HasChange("default_reaction_emoji") {
		patch.DefaultReactionEmoji = &forumDefaultEmoji{}
		hasAny = true
	}

	if v, ok := d.GetOk("default_sort_order"); ok {
		sortOrder := v.(int)
		patch.DefaultSortOrder = &sortOrder
		hasAny = true
	}

	if v, ok := d.GetOk("default_forum_layout"); ok {
		layout := v.(int)
		patch.DefaultForumLayout = &layout
		hasAny = true
	}

	return patch, hasAny
}

func expandForumTags(blocks []interface{}) []forumTag {
	tags := make([]forumTag, 0, len(blocks))
	for _, raw := range blocks {
		block, _ := raw.(map[string]interface{})
		if block == nil {
			continue
		}
		tag := forumTag{
			Name:      block["name"].(string),
			Moderated: block["moderated"].(bool),
		}
		if id, _ := block["id"].(string); id != "" {
			tag.ID = id
		}
		if v, _ := block["emoji_id"].(string); v != "" {
			tag.EmojiID = &v
		}
		if v, _ := block["emoji_name"].(string); v != "" {
			tag.EmojiName = &v
		}
		tags = append(tags, tag)
	}
	return tags
}

func expandForumDefaultEmoji(block map[string]interface{}) *forumDefaultEmoji {
	emoji := &forumDefaultEmoji{}
	if v, _ := block["emoji_id"].(string); v != "" {
		emoji.EmojiID = &v
	}
	if v, _ := block["emoji_name"].(string); v != "" {
		emoji.EmojiName = &v
	}
	return emoji
}

func flattenForumTags(tags []forumTag) []interface{} {
	out := make([]interface{}, 0, len(tags))
	for _, t := range tags {
		entry := map[string]interface{}{
			"id":        t.ID,
			"name":      t.Name,
			"moderated": t.Moderated,
		}
		if t.EmojiID != nil {
			entry["emoji_id"] = *t.EmojiID
		}
		if t.EmojiName != nil {
			entry["emoji_name"] = *t.EmojiName
		}
		out = append(out, entry)
	}
	return out
}

func flattenForumDefaultEmoji(e *forumDefaultEmoji) []interface{} {
	if e == nil || (e.EmojiID == nil && e.EmojiName == nil) {
		return nil
	}
	entry := map[string]interface{}{}
	if e.EmojiID != nil {
		entry["emoji_id"] = *e.EmojiID
	}
	if e.EmojiName != nil {
		entry["emoji_name"] = *e.EmojiName
	}
	return []interface{}{entry}
}

// applyForumChannelPatch PATCHes the channel with forum-specific fields. Only
// fields that have user-configured values (or were just cleared) are sent.
func applyForumChannelPatch(ctx context.Context, client *discordgo.Session, channelID string, d *schema.ResourceData, includeUnset bool) error {
	patch, hasAny := buildForumChannelPatch(d, includeUnset)
	if !hasAny {
		return nil
	}
	endpoint := discordgo.EndpointChannel(channelID)
	_, err := client.RequestWithBucketID("PATCH", endpoint, patch, endpoint, discordgo.WithContext(ctx))
	return err
}

// readForumChannelFields fetches the channel JSON and writes forum-specific
// fields back into the schema. Required because older discordgo Channel
// structs don't expose these fields.
func readForumChannelFields(ctx context.Context, client *discordgo.Session, channelID string, d *schema.ResourceData) error {
	endpoint := discordgo.EndpointChannel(channelID)
	body, err := client.RequestWithBucketID("GET", endpoint, nil, endpoint, discordgo.WithContext(ctx))
	if err != nil {
		return err
	}
	var resp channelReadResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	d.Set("available_tags", flattenForumTags(resp.AvailableTags))
	d.Set("default_reaction_emoji", flattenForumDefaultEmoji(resp.DefaultReactionEmoji))
	if resp.DefaultSortOrder != nil {
		d.Set("default_sort_order", *resp.DefaultSortOrder)
	}
	if resp.DefaultForumLayout != nil {
		d.Set("default_forum_layout", *resp.DefaultForumLayout)
	}
	return nil
}
