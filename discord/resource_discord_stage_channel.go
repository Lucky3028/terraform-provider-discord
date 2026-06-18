package discord

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDiscordStageChannel() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceChannelCreate,
		ReadContext:   resourceChannelRead,
		UpdateContext: resourceChannelUpdate,
		DeleteContext: resourceChannelDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "A resource to create a stage channel.",
		Schema: getChannelSchema("stage", map[string]*schema.Schema{
			"bitrate": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     64000,
				Description: "Bitrate of the channel.",
			},
			"user_limit": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "User limit of the channel.",
			},
		}),
	}
}
