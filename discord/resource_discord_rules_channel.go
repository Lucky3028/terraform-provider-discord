package discord

import (
	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/net/context"
)

func resourceDiscordRulesChannel() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRulesChannelCreate,
		ReadContext:   resourceRulesChannelRead,
		UpdateContext: resourceRulesChannelUpdate,
		DeleteContext: resourceRulesChannelDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Description: "Sets the rules channel of a Community-enabled Discord server (the channel " +
			"where rules/guidelines are displayed). The server must have the Community feature enabled.",
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the server to set the rules channel for.",
			},
			"rules_channel_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the channel that will be used as the rules channel.",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the server.",
			},
		},
	}
}

func resourceRulesChannelCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	serverId := d.Get("server_id").(string)

	server, err := client.Guild(serverId, discordgo.WithContext(ctx))
	if err != nil {
		return diag.Errorf("Failed to find server: %s", err.Error())
	}

	params := &discordgo.GuildParams{
		// Discord only honors rules_channel_id when the COMMUNITY feature is present
		// in the same Modify Guild request, so pass the guild's current features through.
		Features:       server.Features,
		RulesChannelID: d.Get("rules_channel_id").(string),
	}
	if _, err := client.GuildEdit(serverId, params, discordgo.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to edit server: %s", err.Error())
	}

	d.SetId(serverId)

	return diags
}

func resourceRulesChannelRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	serverId := d.Id()

	server, err := client.Guild(serverId, discordgo.WithContext(ctx))
	if err != nil {
		return diag.Errorf("Error fetching server: %s", err.Error())
	}

	d.Set("server_id", serverId)
	d.Set("rules_channel_id", server.RulesChannelID)

	return diags
}

func resourceRulesChannelUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	serverId := d.Get("server_id").(string)
	server, err := client.Guild(serverId, discordgo.WithContext(ctx))
	if err != nil {
		return diag.Errorf("Error fetching server: %s", err.Error())
	}

	if d.HasChange("rules_channel_id") {
		params := &discordgo.GuildParams{
			// rules_channel_id is only honored alongside the COMMUNITY feature.
			Features:       server.Features,
			RulesChannelID: d.Get("rules_channel_id").(string),
		}
		if _, err := client.GuildEdit(serverId, params, discordgo.WithContext(ctx)); err != nil {
			return diag.Errorf("Failed to edit server: %s", err.Error())
		}
	}

	return diags
}

func resourceRulesChannelDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	// Community-enabled servers require a rules channel, so it cannot be unset.
	// Removing this resource just stops Terraform from managing the setting.
	tflog.Warn(ctx, "discord_rules_channel: a Community server's rules channel cannot be unset; "+
		"leaving it in place and removing only from Terraform state.")

	return diags
}
