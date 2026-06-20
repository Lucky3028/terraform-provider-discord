package discord

import (
	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/net/context"
)

func resourceDiscordMemberNickname() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMemberNicknameCreate,
		ReadContext:   resourceMemberNicknameRead,
		UpdateContext: resourceMemberNicknameUpdate,
		DeleteContext: resourceMemberNicknameDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceMemberNicknameImport,
		},

		Description: "A resource to manage a member's server nickname.",
		Schema: map[string]*schema.Schema{
			"user_id": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "ID of the user to manage the nickname for.",
			},
			"server_id": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "ID of the server to manage the nickname in.",
			},
			"nick": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Server nickname to set for the member. Omit or set to an empty string to clear the nickname.",
			},
		},
	}
}

func resourceMemberNicknameImport(ctx context.Context, data *schema.ResourceData, i interface{}) ([]*schema.ResourceData, error) {
	serverId, userId, err := parseTwoIds(data.Id())
	if err != nil {
		return nil, err
	}

	data.Set("server_id", serverId)
	data.Set("user_id", userId)

	return schema.ImportStatePassthroughContext(ctx, data, i)
}

func resourceMemberNicknameCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	d.SetId(generateTwoPartId(d.Get("server_id").(string), d.Get("user_id").(string)))

	return resourceMemberNicknameUpdate(ctx, d, m)
}

func resourceMemberNicknameRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	serverId := d.Get("server_id").(string)
	userId := d.Get("user_id").(string)

	member, err := client.GuildMember(serverId, userId, discordgo.WithContext(ctx))
	if err != nil {
		return diag.Errorf("Could not get member %s in %s: %s", userId, serverId, err.Error())
	}

	d.SetId(generateTwoPartId(serverId, userId))
	d.Set("nick", member.Nick)

	return diags
}

func resourceMemberNicknameUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	serverId := d.Get("server_id").(string)
	userId := d.Get("user_id").(string)
	nick := d.Get("nick").(string)

	if err := client.GuildMemberNickname(serverId, userId, nick, discordgo.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to edit member nickname %s: %s", userId, err.Error())
	}

	d.SetId(generateTwoPartId(serverId, userId))

	diags = append(diags, resourceMemberNicknameRead(ctx, d, m)...)

	return diags
}

func resourceMemberNicknameDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	serverId := d.Get("server_id").(string)
	userId := d.Get("user_id").(string)

	if err := client.GuildMemberNickname(serverId, userId, "", discordgo.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to delete member nickname %s: %s", userId, err.Error())
	}

	d.SetId("")

	return diags
}
