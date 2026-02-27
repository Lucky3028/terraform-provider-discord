package discord

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDiscordSystemChannel() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceDiscordSystemChannelRead,
		Description: "Fetches a server's system channel.",
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The server ID to search for.",
			},
			"system_channel_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the server's system channel.",
			},
			"system_channel_flags": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The system channel flags of the server.",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the server.",
			},
		},
	}
}

func dataSourceDiscordSystemChannelRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	var err error
	var server *discordgo.Guild
	client := m.(*Context).Session

	serverId := d.Get("server_id").(string)
	if server, err = client.Guild(serverId, discordgo.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to fetch server %s: %s", serverId, err.Error())
	} else {
		d.SetId(serverId)
		d.Set("system_channel_id", server.SystemChannelID)
		d.Set("system_channel_flags", int(server.SystemChannelFlags))

		return diags
	}
}
