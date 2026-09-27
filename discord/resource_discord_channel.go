package discord

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/net/context"
)

func getChannelSchema(channelType string, s map[string]*schema.Schema) map[string]*schema.Schema {
	addedSchema := map[string]*schema.Schema{
		"server_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "ID of server this channel is in.",
		},
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The ID of the channel.",
		},
		"channel_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The ID of the channel.",
		},
		"type": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The type of the channel. This is only for internal use and should never be provided.",
			ValidateDiagFunc: func(i interface{}, path cty.Path) (diags diag.Diagnostics) {
				if i.(string) != channelType {
					diags = append(diags, diag.Errorf("type must be %s, %s passed", channelType, i.(string))...)
				}

				return diags
			},
			DefaultFunc: func() (interface{}, error) {
				return channelType, nil
			},
		},
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Name of the channel.",
		},
		"position": {
			Type:     schema.TypeInt,
			Default:  1,
			Optional: true,
			Description: "Position of the channel, `0`-indexed. " +
				"**Deprecated** — Discord normalises channel positions per-type within each category, which makes per-channel position values diverge from what you set in HCL and produces permanent state drift. " +
				"Use `discord_channel_order` instead to manage ordering atomically via Discord's bulk reorder endpoint.",
			Deprecated: "Use the discord_channel_order resource to manage channel ordering atomically. The per-channel position field cannot produce a stable plan on guilds with many channels.",
			ValidateFunc: func(val interface{}, key string) (warns []string, errors []error) {
				v := val.(int)

				if v < 0 {
					errors = append(errors, fmt.Errorf("position must be greater than 0, got: %d", v))
				}

				return
			},
		},
	}

	if channelType != "category" {
		addedSchema["category"] = &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "ID of category to place this channel in.",
		}
		addedSchema["sync_perms_with_category"] = &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Whether channel permissions should be synced with the category this channel is in.",
		}
	}

	for k, v := range s {
		addedSchema[k] = v
	}

	return addedSchema
}

func validateChannel(d *schema.ResourceData) (bool, error) {
	channelType := d.Get("type").(string)

	switch channelType {
	case "category":
		{
			if _, ok := d.GetOk("category"); ok {
				return false, errors.New("category cannot be a child of another category")
			}
			if _, ok := d.GetOk("nsfw"); ok {
				return false, errors.New("nsfw is not allowed on categories")
			}
		}
	case "voice":
		{
			if _, ok := d.GetOk("topic"); ok {
				return false, errors.New("topic is not allowed on voice channels")
			}
			if _, ok := d.GetOk("nsfw"); ok {
				return false, errors.New("nsfw is not allowed on voice channels")
			}
		}
	case "text", "news":
		{
			if _, ok := d.GetOk("bitrate"); ok {
				return false, errors.New("bitrate is not allowed on text channels")
			}
			if _, ok := d.GetOk("user_limit"); ok {
				if d.Get("user_limit").(int) > 0 {
					return false, errors.New("user_limit is not allowed on text channels")
				}
			}
			name := d.Get("name").(string)
			if strings.ToLower(name) != name {
				return false, errors.New("name must be lowercase")
			}
		}
	}

	return true, nil
}

// buildChannelCreateData maps the resource config onto Discord's channel-create
// payload. It is kept apart from resourceChannelCreate so the per-type field
// selection can be tested without a guild to create channels in: the only
// coverage it had was an acceptance test, skipped unless DISCORD_TEST_SERVER_ID
// is set, so a field missing from a branch here went unnoticed in CI.
func buildChannelCreateData(d *schema.ResourceData, channelTypeInt discordgo.ChannelType) discordgo.GuildChannelCreateData {
	channelType := d.Get("type").(string)

	var (
		topic            string
		bitrate          = 64000
		userlimit        int
		nsfw             bool
		parentId         string
		rateLimitPerUser int
	)

	switch channelType {
	// Forums belong here, not in a branch of their own: Discord's forum channel
	// carries a topic — shown as the forum's post guidelines — plus an nsfw flag
	// and slowmode, exactly as text and news channels do.
	case "text", "news", "forum":
		{
			if v, ok := d.GetOk("topic"); ok {
				topic = v.(string)
			}
			if v, ok := d.GetOk("nsfw"); ok {
				nsfw = v.(bool)
			}
			rateLimitPerUser = d.Get("rate_limit_per_user").(int)
		}
	case "voice":
		{
			if v, ok := d.GetOk("bitrate"); ok {
				bitrate = v.(int)
			}
			if v, ok := d.GetOk("user_limit"); ok {
				userlimit = v.(int)
			}
		}
	}

	if channelType != "category" {
		if v, ok := d.GetOk("category"); ok {
			parentId = v.(string)
		}
	}

	return discordgo.GuildChannelCreateData{
		Name:             d.Get("name").(string),
		Type:             channelTypeInt,
		Topic:            topic,
		Bitrate:          bitrate,
		UserLimit:        userlimit,
		Position:         d.Get("position").(int),
		ParentID:         parentId,
		NSFW:             nsfw,
		RateLimitPerUser: rateLimitPerUser,
	}
}

// buildChannelEdit maps the resource config onto Discord's channel-edit payload,
// falling back to the channel's current values for whatever the plan left
// unchanged. Split out for the same reason as buildChannelCreateData.
func buildChannelEdit(d *schema.ResourceData, current *discordgo.Channel) *discordgo.ChannelEdit {
	channelType := d.Get("type").(string)

	var (
		name             string
		position         int
		topic            string
		nsfw             bool
		bitRate          = 64000
		userLimit        int
		parentId         string
		rateLimitPerUser int
	)

	name = map[bool]string{true: d.Get("name").(string), false: current.Name}[d.HasChange("name")]
	position = map[bool]int{true: d.Get("position").(int), false: current.Position}[d.HasChange("position")]

	switch channelType {
	// See buildChannelCreateData for why forums share this branch.
	case "text", "news", "forum":
		{
			topic = map[bool]string{true: d.Get("topic").(string), false: current.Topic}[d.HasChange("topic")]
			nsfw = map[bool]bool{true: d.Get("nsfw").(bool), false: current.NSFW}[d.HasChange("nsfw")]
			rateLimitPerUser = map[bool]int{true: d.Get("rate_limit_per_user").(int), false: current.RateLimitPerUser}[d.HasChange("rate_limit_per_user")]
		}
	case "voice":
		{
			bitRate = map[bool]int{true: d.Get("bitrate").(int), false: current.Bitrate}[d.HasChange("bitrate")]
			userLimit = map[bool]int{true: d.Get("user_limit").(int), false: current.UserLimit}[d.HasChange("user_limit")]
		}
	}

	if channelType != "category" && d.HasChange("category") {
		id := d.Get("category").(string)
		parentId = map[bool]string{true: id, false: ""}[d.Get("category").(string) != ""]
	}

	return &discordgo.ChannelEdit{
		Name:             name,
		Position:         &position,
		Topic:            topic,
		NSFW:             &nsfw,
		Bitrate:          bitRate,
		UserLimit:        userLimit,
		ParentID:         parentId,
		RateLimitPerUser: &rateLimitPerUser,
	}
}

func resourceChannelCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	if ok, reason := validateChannel(d); !ok {
		return diag.FromErr(reason)
	}

	serverId := d.Get("server_id").(string)
	channelType := d.Get("type").(string)
	channelTypeInt, okay := getDiscordChannelType(channelType)
	if !okay {
		return diag.Errorf("Invalid channel type: %s", channelType)
	}

	isCategoryCh := channelType == "category"

	channel, err := client.GuildChannelCreateComplex(
		serverId,
		buildChannelCreateData(d, channelTypeInt),
		discordgo.WithContext(ctx),
	)

	if err != nil {
		return diag.Errorf("Failed to create channel: %s", err.Error())
	}

	d.SetId(channel.ID)
	d.Set("server_id", serverId)
	d.Set("channel_id", channel.ID)

	if channelType == "forum" {
		if err := applyForumChannelPatch(ctx, client, channel.ID, d, false); err != nil {
			return append(diags, diag.Errorf("Failed to apply forum channel config to %s: %s", channel.ID, err.Error())...)
		}
	}

	if !isCategoryCh {
		if v, ok := d.GetOk("sync_perms_with_category"); ok && v.(bool) {
			if channel.ParentID == "" {
				return append(diags, diag.Errorf("Can't sync permissions with category. Channel (%s) doesn't have a category", channel.ID)...)
			}
			parent, err := client.Channel(channel.ParentID)
			if err != nil {
				return append(diags, diag.Errorf("Can't sync permissions with category. Channel (%s) doesn't have a category", channel.ID)...)
			}

			if err = syncChannelPermissions(client, ctx, parent, channel); err != nil {
				return append(diags, diag.Errorf("Can't sync permissions with category: %s", channel.ID)...)
			}
		}
	}

	return diags
}

func resourceChannelRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	channel, err := client.Channel(d.Id(), discordgo.WithContext(ctx))
	if err != nil {
		return diag.Errorf("Failed to fetch channel %s: %s", d.Id(), err.Error())
	}

	channelType, ok := getTextChannelType(channel.Type)
	if !ok {
		return diag.Errorf("Invalid channel type: %d", channel.Type)
	}

	d.Set("server_id", channel.GuildID)
	d.Set("type", channelType)
	d.Set("name", channel.Name)
	d.Set("position", channel.Position)

	switch channelType {
	case "text", "news":
		{
			d.Set("topic", channel.Topic)
			d.Set("nsfw", channel.NSFW)
			d.Set("rate_limit_per_user", channel.RateLimitPerUser)
		}
	case "forum":
		{
			d.Set("topic", channel.Topic)
			d.Set("nsfw", channel.NSFW)
			d.Set("rate_limit_per_user", channel.RateLimitPerUser)
			if err := readForumChannelFields(ctx, client, channel.ID, d); err != nil {
				return diag.Errorf("Failed to read forum channel config for %s: %s", channel.ID, err.Error())
			}
		}
	case "voice":
		{
			d.Set("bitrate", channel.Bitrate)
			d.Set("user_limit", channel.UserLimit)
		}
	}

	if channelType != "category" {
		if channel.ParentID == "" {
			d.Set("sync_perms_with_category", false)
		} else {
			parent, err := client.Channel(channel.ParentID)
			if err != nil {
				return diag.Errorf("Failed to fetch category of channel %s: %s", channel.ID, err.Error())
			}

			synced := arePermissionsSynced(channel, parent)
			d.Set("sync_perms_with_category", synced)
		}
	}

	if channel.ParentID == "" {
		d.Set("category", nil)
	} else {
		d.Set("category", channel.ParentID)
	}

	return diags
}

func resourceChannelUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session
	if ok, reason := validateChannel(d); !ok {
		return diag.FromErr(reason)
	}

	channel, _ := client.Channel(d.Id(), discordgo.WithContext(ctx))
	channelType := d.Get("type").(string)

	channel, err := client.ChannelEditComplex(
		d.Id(),
		buildChannelEdit(d, channel),
		discordgo.WithContext(ctx),
	)
	if err != nil {
		return diag.Errorf("Failed to update channel %s: %s", d.Id(), err.Error())
	}

	if channelType == "forum" {
		if err := applyForumChannelPatch(ctx, client, d.Id(), d, true); err != nil {
			return diag.Errorf("Failed to apply forum channel config to %s: %s", d.Id(), err.Error())
		}
	}

	if channelType != "category" {
		if v, ok := d.GetOk("sync_perms_with_category"); ok && v.(bool) {
			if channel.ParentID == "" {
				return append(diags, diag.Errorf("Can't sync permissions with category. Channel (%s) doesn't have a category", channel.ID)...)
			}
			parent, err := client.Channel(channel.ParentID, discordgo.WithContext(ctx))
			if err != nil {
				return append(diags, diag.Errorf("Can't sync permissions with category. Channel (%s) doesn't have a category", channel.ID)...)
			}

			if err = syncChannelPermissions(client, ctx, parent, channel); err != nil {
				return append(diags, diag.Errorf("Can't sync permissions with category: %s", channel.ID)...)
			}
		}
	}

	return diags
}

func resourceChannelDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	_, err := client.ChannelDelete(d.Id(), discordgo.WithContext(ctx))
	if err != nil {
		return diag.Errorf("Failed to delete channel %s: %s", d.Id(), err.Error())
	}

	return diags
}
