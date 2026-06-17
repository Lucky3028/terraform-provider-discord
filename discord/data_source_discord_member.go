package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/net/context"
)

func dataSourceDiscordMember() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceMemberRead,
		Description: "Fetches a member's information from a server.",

		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The server ID to search for the user in.",
			},
			"user_id": {
				ExactlyOneOf: []string{"user_id", "username"},
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The user ID to search for. Required if not searching by `username` / `discriminator`.",
			},
			"username": {
				ExactlyOneOf: []string{"user_id", "username"},
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The username to search for.",
			},
			"discriminator": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The discriminator to search for. `username` is required when using this.",
				Deprecated:  "Discriminator is being deprecated by Discord. Only use this if there are users who haven't migrated their username.",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The user's ID.",
			},
			"joined_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The time at which the user joined.",
			},
			"premium_since": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The time at which the user became premium.",
			},
			"avatar": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The avatar hash of the user.",
			},
			"nick": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current nickname of the user.",
			},
			"roles": {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Set:         schema.HashString,
				Description: "IDs of the roles that the user has.",
			},
			"in_server": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the user is in the server.",
			},
		},
	}
}

func dataSourceMemberRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	var member *discordgo.Member
	var memberErr error
	client := m.(*Context).Session
	serverId := d.Get("server_id").(string)

	if v, ok := d.GetOk("user_id"); ok {

		member, memberErr = client.GuildMember(serverId, v.(string), discordgo.WithContext(ctx))
	}

	if v, ok := d.GetOk("username"); ok {
		username := v.(string)
		members, err := client.GuildMembersSearch(serverId, username, 1000, discordgo.WithContext(ctx))
		if err != nil {
			return diag.Errorf("Failed to fetch members for %s: %s", serverId, err.Error())
		}

		discriminator := d.Get("discriminator").(string)
		if member = findMemberByUsername(members, username, discriminator); member == nil {
			memberErr = fmt.Errorf("failed to find member with username %q in server %s", username, serverId)
		}
	}
	if memberErr != nil {
		return diag.FromErr(memberErr)
	}
	d.Set("in_server", memberErr == nil)
	if memberErr != nil {
		d.Set("joined_at", nil)
		d.Set("premium_since", nil)
		d.Set("roles", nil)
		d.Set("username", nil)
		d.Set("discriminator", nil)
		d.Set("avatar", nil)
		d.Set("nick", nil)
		return diags
	}

	roles := make([]string, 0, len(member.Roles))
	for _, r := range member.Roles {
		roles = append(roles, r)
	}
	if member.PremiumSince == nil {
		d.Set("premium_since", nil)
	}

	d.SetId(member.User.ID)
	d.Set("joined_at", member.JoinedAt.String())
	d.Set("roles", roles)
	d.Set("username", member.User.Username)
	d.Set("discriminator", member.User.Discriminator)
	d.Set("avatar", member.User.Avatar)
	d.Set("nick", member.Nick)

	return diags
}

// findMemberByUsername returns the first member whose username matches, or nil
// if none do. The discriminator is only enforced for un-migrated legacy
// accounts: migrated accounts report a discriminator of "0", while the optional
// field defaults to "" — neither should block a username match.
func findMemberByUsername(members []*discordgo.Member, username, discriminator string) *discordgo.Member {
	for _, m := range members {
		if m.User.Username != username {
			continue
		}
		if discriminator != "" && discriminator != "0" && m.User.Discriminator != discriminator {
			continue
		}
		return m
	}
	return nil
}
