package discord

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var permissions map[string]int64

// Reference: https://discord.com/developers/docs/topics/permissions
func dataSourceDiscordPermission() *schema.Resource {
	permissions = map[string]int64{
		"create_instant_invite":       0x1,
		"kick_members":                0x2,
		"ban_members":                 0x4,
		"administrator":               0x8,
		"manage_channels":             0x10,
		"manage_guild":                0x20,
		"add_reactions":               0x40,
		"view_audit_log":              0x80,
		"priority_speaker":            0x100,
		"stream":                      0x200,
		"view_channel":                0x400,
		"send_messages":               0x800,
		"send_tts_messages":           0x1000,
		"manage_messages":             0x2000,
		"embed_links":                 0x4000,
		"attach_files":                0x8000,
		"read_message_history":        0x10000,
		"mention_everyone":            0x20000,
		"use_external_emojis":         0x40000,
		"view_guild_insights":         0x80000,
		"connect":                     0x100000,
		"speak":                       0x200000,
		"mute_members":                0x400000,
		"deafen_members":              0x800000,
		"move_members":                0x1000000,
		"use_vad":                     0x2000000,
		"change_nickname":             0x4000000,
		"manage_nicknames":            0x8000000,
		"manage_roles":                0x10000000,
		"manage_webhooks":             0x20000000,
		"manage_emojis":               0x40000000,
		"use_application_commands":    0x80000000,
		"request_to_speak":            0x100000000,
		"manage_events":               0x200000000,
		"manage_threads":              0x400000000,
		"create_public_threads":       0x800000000,
		"create_private_threads":      0x1000000000,
		"use_external_stickers":       0x2000000000,
		"send_thread_messages":        0x4000000000,
		"start_embedded_activities":   0x8000000000,
		"moderate_members":            0x10000000000,
		"view_monetization_analytics": 0x20000000000,
		"use_soundboard":              0x40000000000,
		"create_expressions":          0x80000000000,
		"create_events":               0x100000000000,
		"use_external_sounds":         0x200000000000,
		"send_voice_messages":         0x400000000000,
		"set_voice_channel_status":    0x1000000000000,
		"send_polls":                  0x2000000000000,
		"use_external_apps":           0x4000000000000,
		"pin_messages":                0x8000000000000,
		"bypass_slowmode":             0x10000000000000,
	}

	schemaMap := make(map[string]*schema.Schema)
	schemaMap["allow_extends"] = &schema.Schema{
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "The base permission bits for allow to extend.",
	}
	schemaMap["deny_extends"] = &schema.Schema{
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "The base permission bits for deny to extend.",
	}
	schemaMap["allow_bits"] = &schema.Schema{
		Type:        schema.TypeInt,
		Computed:    true,
		Description: "The allow permission bits.",
	}
	schemaMap["deny_bits"] = &schema.Schema{
		Type:        schema.TypeInt,
		Computed:    true,
		Description: "The deny permission bits.",
	}
	for k := range permissions {
		schemaMap[k] = &schema.Schema{
			Optional:     true,
			Type:         schema.TypeString,
			Default:      "unset",
			Description:  fmt.Sprintf("The value to set for the `%s` permission bit. Must be `allow`, `unset`, or `deny`. (default `unset`)", k),
			ValidateFunc: validation.StringInSlice([]string{"allow", "unset", "deny"}, false),
		}
	}

	return &schema.Resource{
		ReadContext: dataSourceDiscordPermissionRead,
		Description: "A simple helper to get computed bit total of a list of permissions.",
		Schema:      schemaMap,
	}
}

func dataSourceDiscordPermissionRead(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	var allowBits int64
	var denyBits int64
	for perm, bit := range permissions {
		switch d.Get(perm).(string) {
		case "allow":
			allowBits |= bit
		case "deny":
			denyBits |= bit
		}
	}

	d.SetId(strconv.Itoa(Hashcode(fmt.Sprintf("%d:%d", allowBits, denyBits))))
	d.Set("allow_bits", allowBits|(int64(d.Get("allow_extends").(int))))
	d.Set("deny_bits", denyBits|(int64(d.Get("deny_extends").(int))))

	return diags
}
