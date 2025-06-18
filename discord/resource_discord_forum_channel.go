package discord

import (
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
		}),
	}
}
