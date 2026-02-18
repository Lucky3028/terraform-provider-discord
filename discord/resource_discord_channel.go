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
			Type:        schema.TypeInt,
			Default:     1,
			Optional:    true,
			Description: "Position of the channel, `0`-indexed.",
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

	var (
		topic     string
		bitrate   = 64000
		userlimit int
		nsfw      bool
		parentId  string
	)

	switch channelType {
	case "text", "news":
		{
			if v, ok := d.GetOk("topic"); ok {
				topic = v.(string)
			}
			if v, ok := d.GetOk("nsfw"); ok {
				nsfw = v.(bool)
			}
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

	isCategoryCh := channelType == "category"

	if !isCategoryCh {
		if v, ok := d.GetOk("category"); ok {
			parentId = v.(string)
		}
	}
	channel, err := client.GuildChannelCreateComplex(serverId, discordgo.GuildChannelCreateData{
		Name:      d.Get("name").(string),
		Type:      channelTypeInt,
		Topic:     topic,
		Bitrate:   bitrate,
		UserLimit: userlimit,
		Position:  d.Get("position").(int),
		ParentID:  parentId,
		NSFW:      nsfw,
	}, discordgo.WithContext(ctx))

	if err != nil {
		return diag.Errorf("Failed to create channel: %s", err.Error())
	}

	d.SetId(channel.ID)
	d.Set("server_id", serverId)
	d.Set("channel_id", channel.ID)

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
	case "text":
		{
			d.Set("topic", channel.Topic)
			d.Set("nsfw", channel.NSFW)
		}
	case "news":
		d.Set("topic", channel.Topic)
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

	var (
		name      string
		position  int
		topic     string
		nsfw      bool
		bitRate   = 64000
		userLimit int
		parentId  string
	)

	name = map[bool]string{true: d.Get("name").(string), false: channel.Name}[d.HasChange("name")]
	position = map[bool]int{true: d.Get("position").(int), false: channel.Position}[d.HasChange("position")]

	switch channelType {
	case "text", "news":
		{
			topic = map[bool]string{true: d.Get("topic").(string), false: channel.Topic}[d.HasChange("topic")]
			nsfw = map[bool]bool{true: d.Get("nsfw").(bool), false: channel.NSFW}[d.HasChange("nsfw")]
		}
	case "voice":
		{
			bitRate = map[bool]int{true: d.Get("bitrate").(int), false: channel.Bitrate}[d.HasChange("bitrate")]
			userLimit = map[bool]int{true: d.Get("user_limit").(int), false: channel.UserLimit}[d.HasChange("user_limit")]
		}
	}

	if channelType != "category" && d.HasChange("category") {
		id := d.Get("category").(string)
		parentId = map[bool]string{true: id, false: ""}[d.Get("category").(string) != ""]
	}
	channel, err := client.ChannelEditComplex(d.Id(), &discordgo.ChannelEdit{
		Name:      name,
		Position:  &position,
		Topic:     topic,
		NSFW:      &nsfw,
		Bitrate:   bitRate,
		UserLimit: userLimit,
		ParentID:  parentId,
	}, discordgo.WithContext(ctx))
	if err != nil {
		return diag.Errorf("Failed to update channel %s: %s", d.Id(), err.Error())
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
