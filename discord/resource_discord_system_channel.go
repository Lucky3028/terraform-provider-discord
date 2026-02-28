package discord

import (
	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"golang.org/x/net/context"
)

func resourceDiscordSystemChannel() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSystemChannelCreate,
		ReadContext:   resourceSystemChannelRead,
		UpdateContext: resourceSystemChannelUpdate,
		DeleteContext: resourceSystemChannelDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Description: "Manage the system channel of a Discord server.",
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the server to manage the system channel for.",
			},
			"system_channel_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the channel that will be used as the system channel.",
			},
			"system_channel_flags": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				Description: "System channel flags. Bitwise OR of: " +
					"suppress member join notifications (`1`), " +
					"suppress premium subscriptions (`2`), " +
					"suppress server setup tips (`4`), " +
					"suppress join notification sticker replies (`8`).",
				ValidateFunc: validation.IntBetween(0, 15),
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the server.",
			},
		},
	}
}

func resourceSystemChannelCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	serverId := d.Get("server_id").(string)

	server, err := client.Guild(serverId, discordgo.WithContext(ctx))
	if err != nil {
		return diag.Errorf("Failed to find server: %s", err.Error())
	}

	systemChannelId := server.SystemChannelID
	if v, ok := d.GetOk("system_channel_id"); ok {
		systemChannelId = v.(string)

	} else {
		return diag.Errorf("Failed to parse system channel id")
	}

	params := &discordgo.GuildParams{
		SystemChannelID: systemChannelId,
	}
	if v, ok := d.GetOk("system_channel_flags"); ok {
		params.SystemChannelFlags = discordgo.SystemChannelFlag(v.(int))
	}

	if _, err := client.GuildEdit(serverId, params, discordgo.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to edit server: %s", err.Error())
	}

	d.SetId(serverId)

	return diags
}

func resourceSystemChannelRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	serverId := d.Id()

	server, err := client.Guild(serverId, discordgo.WithContext(ctx))
	if err != nil {
		return diag.Errorf("Error fetching server: %s", err.Error())
	}

	d.Set("server_id", serverId)
	d.Set("system_channel_id", server.SystemChannelID)
	d.Set("system_channel_flags", int(server.SystemChannelFlags))

	return diags
}

func resourceSystemChannelUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	serverId := d.Get("server_id").(string)
	_, err := client.Guild(serverId, discordgo.WithContext(ctx))

	if err != nil {
		return diag.Errorf("Error fetching server: %s", err.Error())
	}

	params := &discordgo.GuildParams{}
	edit := false

	if d.HasChange("system_channel_id") {
		params.SystemChannelID = d.Get("system_channel_id").(string)
		edit = true
	}

	if d.HasChange("system_channel_flags") {
		params.SystemChannelFlags = discordgo.SystemChannelFlag(d.Get("system_channel_flags").(int))
		edit = true
	}

	if edit {
		if _, err := client.GuildEdit(serverId, params, discordgo.WithContext(ctx)); err != nil {
			return diag.Errorf("Failed to edit server: %s", err.Error())
		}
	}

	return diags
}

func resourceSystemChannelDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	serverID := d.Get("server_id").(string)

	if _, err := client.GuildEdit(serverID, &discordgo.GuildParams{
		SystemChannelID: "",
	}, discordgo.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to edit server: %s: %s", serverID, err.Error())
	}

	return diags
}
