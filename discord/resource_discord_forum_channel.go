package discord

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDiscordForumChannel() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceChannelCreate,
		ReadContext:   resourceChannelRead,
		UpdateContext: resourceChannelUpdate,
		DeleteContext: resourceChannelDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "A resource to create a forum channel.",
		Schema: getChannelSchema("forum", map[string]*schema.Schema{
			"topic": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Topic of the channel.",
			},
			"nsfw": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether the channel is NSFW.",
			},
			"available_tags": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: "Tags that can be applied to threads in this forum. Discord allows up to 20 tags per channel.",
				MaxItems:    20,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Discord-assigned ID for the tag.",
						},
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Display name of the tag.",
						},
						"moderated": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "If `true`, only moderators (`MANAGE_THREADS` permission) can apply or remove this tag.",
						},
						"emoji_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "ID of a custom guild emoji. Mutually exclusive with `emoji_name`.",
						},
						"emoji_name": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Unicode emoji character. Mutually exclusive with `emoji_id`.",
						},
					},
				},
			},
			"default_reaction_emoji": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Default emoji used for the new-post reaction button in the forum.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"emoji_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "ID of a custom guild emoji. Mutually exclusive with `emoji_name`.",
						},
						"emoji_name": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Unicode emoji character. Mutually exclusive with `emoji_id`.",
						},
					},
				},
			},
			"default_sort_order": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Default order to sort posts in the forum. `0` = recent activity (Discord default), `1` = creation date.",
				ValidateFunc: func(val interface{}, key string) (warns []string, errors []error) {
					v := val.(int)
					if v < 0 || v > 1 {
						errors = append(errors, fmt.Errorf("%s must be 0 (latest activity) or 1 (creation date), got: %d", key, v))
					}
					return
				},
			},
			"default_forum_layout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "Default layout for the forum. `0` = not set, `1` = list view, `2` = gallery view.",
				ValidateFunc: func(val interface{}, key string) (warns []string, errors []error) {
					v := val.(int)
					if v < 0 || v > 2 {
						errors = append(errors, fmt.Errorf("%s must be 0 (not set), 1 (list view), or 2 (gallery view), got: %d", key, v))
					}
					return
				},
			},
		}),
	}
}
