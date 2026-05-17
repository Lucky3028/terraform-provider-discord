package discord

import (
	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/net/context"
)

func resourceDiscordServerWidget() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceServerWidgetCreate,
		ReadContext:   resourceServerWidgetRead,
		UpdateContext: resourceServerWidgetUpdate,
		DeleteContext: resourceServerWidgetDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Description: "Manage the widget settings of a Discord server.",
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the server to manage the widget for.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether the server widget is enabled.",
			},
			"channel_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The channel ID that the widget will generate an invite to, or null if set to no invite.",
			},
		},
	}
}

func resourceServerWidgetCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	serverId := d.Get("server_id").(string)
	enabled := d.Get("enabled").(bool)

	data := &discordgo.GuildEmbed{
		Enabled: &enabled,
	}
	if v, ok := d.GetOk("channel_id"); ok {
		data.ChannelID = v.(string)
	}

	if err := client.GuildEmbedEdit(serverId, data, discordgo.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to update server widget: %s", err.Error())
	}

	d.SetId(serverId)

	return diags
}

func resourceServerWidgetRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	serverId := d.Id()

	widget, err := client.GuildEmbed(serverId, discordgo.WithContext(ctx))
	if err != nil {
		return diag.Errorf("Failed to fetch server widget: %s", err.Error())
	}

	d.Set("server_id", serverId)
	if widget.Enabled != nil {
		d.Set("enabled", *widget.Enabled)
	}
	d.Set("channel_id", widget.ChannelID)

	return diags
}

func resourceServerWidgetUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	serverId := d.Get("server_id").(string)

	if d.HasChanges("enabled", "channel_id") {
		enabled := d.Get("enabled").(bool)
		data := &discordgo.GuildEmbed{
			Enabled: &enabled,
		}
		if v, ok := d.GetOk("channel_id"); ok {
			data.ChannelID = v.(string)
		}

		if err := client.GuildEmbedEdit(serverId, data, discordgo.WithContext(ctx)); err != nil {
			return diag.Errorf("Failed to update server widget: %s", err.Error())
		}
	}

	return diags
}

func resourceServerWidgetDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	serverId := d.Get("server_id").(string)
	disabled := false

	if err := client.GuildEmbedEdit(serverId, &discordgo.GuildEmbed{
		Enabled: &disabled,
	}, discordgo.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to disable server widget: %s", err.Error())
	}

	return diags
}
