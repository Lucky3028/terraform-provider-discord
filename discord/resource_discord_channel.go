package discord

import (
	"errors"
	"fmt"
	"strings"

	"github.com/andersfylling/disgord"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/net/context"
)

func getChannelSchema(channelType string, s map[string]*schema.Schema) map[string]*schema.Schema {
	addedSchema := map[string]*schema.Schema{
		"server_id": {
			Type:     schema.TypeString,
			Required: true,
		},
		"type": {
			Type:     schema.TypeString,
			Required: true,
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
			Type:     schema.TypeString,
			Required: true,
		},
		"position": {
			Type:     schema.TypeInt,
			Default:  1,
			Optional: true,
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
			Type:     schema.TypeString,
			Optional: true,
		}
		addedSchema["sync_perms_with_category"] = &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
			Default:  true,
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
	client := m.(*Context).Client

	if ok, reason := validateChannel(d); !ok {
		return diag.FromErr(reason)
	}

	serverId := getMajorId(d.Get("server_id"))
	channelType := d.Get("type").(string)
	channelTypeInt, _ := getDiscordChannelType(channelType)

	var (
		topic     string
		bitrate   uint = 64000
		userlimit uint
		nsfw      bool
		parentId  disgord.Snowflake
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
				bitrate = uint(v.(int))
			}
			if v, ok := d.GetOk("user_limit"); ok {
				userlimit = uint(v.(int))
			}
		}
	}

	isCategoryCh := channelType == "category"

	if !isCategoryCh {
		if v, ok := d.GetOk("category"); ok {
			parentId = getId(v.(string))
		}
	}

	channel, err := client.Guild(serverId).CreateChannel(d.Get("name").(string), &disgord.CreateGuildChannel{
		Type:      channelTypeInt,
		Topic:     topic,
		Bitrate:   bitrate,
		UserLimit: userlimit,
		ParentID:  parentId,
		NSFW:      nsfw,
		Position:  d.Get("position").(int),
	})

	if err != nil {
		return diag.Errorf("Failed to create channel: %s", err.Error())
	}

	d.SetId(channel.ID.String())
	d.Set("server_id", serverId)
	d.Set("channel_id", channel.ID.String())

	if !isCategoryCh {
		if v, ok := d.GetOk("sync_perms_with_category"); ok && v.(bool) {
			if channel.ParentID.IsZero() {
				return append(diags, diag.Errorf("Can't sync permissions with category. Channel (%s) doesn't have a category", channel.ID.String())...)
			}
			parent, err := client.Channel(channel.ParentID).Get()
			if err != nil {
				return append(diags, diag.Errorf("Can't sync permissions with category. Channel (%s) doesn't have a category", channel.ID.String())...)
			}

			if err = syncChannelPermissions(client, ctx, parent, channel); err != nil {
				return append(diags, diag.Errorf("Can't sync permissions with category: %s", channel.ID.String())...)
			}
		}
	}

	return diags
}

func resourceChannelRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Client

	channel, err := client.Channel(getId(d.Id())).Get()
	if err != nil {
		return diag.Errorf("Failed to fetch channel %s: %s", d.Id(), err.Error())
	}

	channelType, ok := getTextChannelType(channel.Type)
	if !ok {
		return diag.Errorf("Invalid channel type: %d", channel.Type)
	}

	d.Set("server_id", channel.GuildID.String())
	d.Set("type", channelType)
	d.Set("name", channel.Name)
	d.Set("position", channel.Position)

	switch channelType {
	case "text", "news":
		{
			d.Set("topic", channel.Topic)
			d.Set("nsfw", channel.NSFW)
		}
	case "voice":
		{
			d.Set("bitrate", channel.Bitrate)
			d.Set("user_limit", channel.UserLimit)
		}
	}

	if channelType != "category" {
		if channel.ParentID.IsZero() {
			d.Set("sync_perms_with_category", false)
		} else {
			parent, err := client.Channel(channel.ParentID).Get()
			if err != nil {
				return diag.Errorf("Failed to fetch category of channel %s: %s", channel.ID.String(), err.Error())
			}

			synced := arePermissionsSynced(channel, parent)
			d.Set("sync_perms_with_category", synced)
		}
	}

	if channel.ParentID.IsZero() {
		d.Set("category", nil)
	} else {
		d.Set("category", channel.ParentID.String())
	}

	return diags
}

func resourceChannelUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Client
	if ok, reason := validateChannel(d); !ok {
		return diag.FromErr(reason)
	}

	channel, _ := client.Channel(getId(d.Id())).Get()
	channelType := d.Get("type").(string)

	var (
		name      string
		position  uint
		topic     string
		nsfw      bool
		bitRate   uint = 64000
		userLimit uint
		parentId  *disgord.Snowflake
	)

	name = map[bool]string{true: d.Get("name").(string), false: channel.Name}[d.HasChange("name")]
	position = map[bool]uint{true: uint(d.Get("position").(int)), false: uint(channel.Position)}[d.HasChange("position")]

	switch channelType {
	case "text", "news":
		{
			topic = map[bool]string{true: d.Get("topic").(string), false: channel.Topic}[d.HasChange("topic")]
			nsfw = map[bool]bool{true: d.Get("nsfw").(bool), false: channel.NSFW}[d.HasChange("nsfw")]
		}
	case "voice":
		{
			bitRate = map[bool]uint{true: uint(d.Get("bitrate").(int)), false: channel.Bitrate}[d.HasChange("bitrate")]
			userLimit = map[bool]uint{true: uint(d.Get("user_limit").(int)), false: channel.UserLimit}[d.HasChange("user_limit")]
		}
	}

	if channelType != "category" && d.HasChange("category") {
		id := getId(d.Get("category").(string))
		parentId = map[bool]*disgord.Snowflake{true: &id, false: nil}[d.Get("category").(string) != ""]
	}

	channel, err := client.Channel(channel.ID).Update(&disgord.UpdateChannel{
		Name:      &name,
		Position:  &position,
		Topic:     &topic,
		NSFW:      &nsfw,
		Bitrate:   &bitRate,
		UserLimit: &userLimit,
		ParentID:  parentId,
	})
	if err != nil {
		return diag.Errorf("Failed to update channel %s: %s", d.Id(), err.Error())
	}

	if channelType != "category" {
		if v, ok := d.GetOk("sync_perms_with_category"); ok && v.(bool) {
			if channel.ParentID.IsZero() {
				return append(diags, diag.Errorf("Can't sync permissions with category. Channel (%s) doesn't have a category", channel.ID.String())...)
			}
			parent, err := client.Channel(channel.ParentID).Get()
			if err != nil {
				return append(diags, diag.Errorf("Can't sync permissions with category. Channel (%s) doesn't have a category", channel.ID.String())...)
			}

			if err = syncChannelPermissions(client, ctx, parent, channel); err != nil {
				return append(diags, diag.Errorf("Can't sync permissions with category: %s", channel.ID.String())...)
			}
		}
	}

	return diags
}

func resourceChannelDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Client

	_, err := client.Channel(getId(d.Id())).Delete()
	if err != nil {
		return diag.Errorf("Failed to delete channel %s: %s", d.Id(), err.Error())
	}

	return diags
}
