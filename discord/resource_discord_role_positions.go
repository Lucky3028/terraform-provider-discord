package discord

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDiscordRolePositions() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRolePositionsUpsert,
		ReadContext:   resourceRolePositionsRead,
		UpdateContext: resourceRolePositionsUpsert,
		DeleteContext: resourceRolePositionsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Atomically sets the positions of multiple roles in a single Discord API call. Use this when ordering many roles consistently — Discord's per-role position updates (set via `discord_role.position`) can race against each other and produce drift when several roles change simultaneously. Roles not listed here are left untouched. Discord's role hierarchy is bot-relative: roles above the bot's own role cannot be moved.",
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the guild whose roles to reorder.",
			},
			"position": {
				Type:        schema.TypeSet,
				Required:    true,
				MinItems:    1,
				Description: "One block per role whose position should be managed. Roles not listed are left untouched.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "ID of the role.",
						},
						"position": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Target position. Discord's `@everyone` role is always at position 0; higher numbers appear higher in the role list.",
							ValidateFunc: func(val interface{}, key string) (warns []string, errors []error) {
								v := val.(int)
								if v < 0 {
									errors = append(errors, fmt.Errorf("%s must be >= 0, got %d", key, v))
								}
								return
							},
						},
					},
				},
				Set: rolePositionHash,
			},
		},
	}
}

func rolePositionHash(v interface{}) int {
	m := v.(map[string]interface{})
	id, _ := m["role_id"].(string)
	return schema.HashString(id)
}

type rolePositionPayload struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

func resourceRolePositionsUpsert(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Context).Session
	serverID := d.Get("server_id").(string)

	positions := expandRolePositions(d.Get("position").(*schema.Set))
	endpoint := discordgo.EndpointGuildRoles(serverID)
	if _, err := client.RequestWithBucketID("PATCH", endpoint, positions, endpoint, discordgo.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to reorder roles in guild %s: %s", serverID, err.Error())
	}

	d.SetId(serverID)
	return resourceRolePositionsRead(ctx, d, m)
}

func resourceRolePositionsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Context).Session
	serverID := d.Id()

	roles, err := client.GuildRoles(serverID, discordgo.WithContext(ctx))
	if err != nil {
		if rest, ok := err.(*discordgo.RESTError); ok && rest.Response != nil && rest.Response.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.Errorf("Failed to fetch roles in guild %s: %s", serverID, err.Error())
	}

	// Only refresh positions for roles the user is managing here. Roles not
	// listed in config are intentionally untouched, so we don't add them.
	managed := managedRoleIDs(d.Get("position").(*schema.Set))
	updated := make([]interface{}, 0, len(managed))
	for _, role := range roles {
		if _, ok := managed[role.ID]; !ok {
			continue
		}
		updated = append(updated, map[string]interface{}{
			"role_id":  role.ID,
			"position": role.Position,
		})
	}
	d.Set("server_id", serverID)
	d.Set("position", updated)
	return nil
}

func resourceRolePositionsDelete(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	// Removing this resource does not restore any "previous" order — Discord
	// has no such concept. Positions stay wherever they currently are; we
	// just stop managing them.
	d.SetId("")
	return nil
}

func expandRolePositions(set *schema.Set) []rolePositionPayload {
	list := set.List()
	out := make([]rolePositionPayload, 0, len(list))
	for _, raw := range list {
		m, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		out = append(out, rolePositionPayload{
			ID:       m["role_id"].(string),
			Position: m["position"].(int),
		})
	}
	return out
}

func managedRoleIDs(set *schema.Set) map[string]struct{} {
	list := set.List()
	out := make(map[string]struct{}, len(list))
	for _, raw := range list {
		m, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if id, _ := m["role_id"].(string); id != "" {
			out[id] = struct{}{}
		}
	}
	return out
}

