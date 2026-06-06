package discord

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDiscordChannelOrder() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceChannelOrderUpsert,
		ReadContext:   resourceChannelOrderRead,
		UpdateContext: resourceChannelOrderUpsert,
		DeleteContext: resourceChannelOrderDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceChannelOrderImport,
		},
		Description: "Atomically orders a set of channels within a Discord guild (or within a single category) using the bulk reorder endpoint `PATCH /guilds/{guild_id}/channels`. " +
			"Discord normalises per-channel `position` values per-type within each category, which makes the field difficult to use predictably. This resource sidesteps that by sending all positions in one call and tracking the result as an ordered list of IDs.",
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the guild whose channels to reorder.",
			},
			"category_id": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "ID of the category whose direct children to order. Omit to order top-level channels and categories (channels with no `parent_id`).",
			},
			"channel_ids": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "Ordered list of channel IDs. Index 0 is the topmost channel. Only channels (and categories, for top-level orderings) listed here are reordered; siblings not in the list are left untouched.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

type channelOrderPatch struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

func resourceChannelOrderUpsert(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Context).Session
	serverID := d.Get("server_id").(string)
	channelIDs, err := expandChannelOrderIDs(d.Get("channel_ids").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}

	payload := make([]channelOrderPatch, 0, len(channelIDs))
	for i, id := range channelIDs {
		payload = append(payload, channelOrderPatch{ID: id, Position: i})
	}

	endpoint := discordgo.EndpointGuildChannels(serverID)
	if _, err := client.RequestWithBucketID("PATCH", endpoint, payload, endpoint, discordgo.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to reorder channels in guild %s: %s", serverID, err.Error())
	}

	d.SetId(buildChannelOrderID(serverID, d.Get("category_id").(string)))
	return resourceChannelOrderRead(ctx, d, m)
}

func resourceChannelOrderRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Context).Session
	serverID, categoryID, err := parseChannelOrderID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	channels, err := client.GuildChannels(serverID, discordgo.WithContext(ctx))
	if err != nil {
		if rest, ok := err.(*discordgo.RESTError); ok && rest.Response != nil && rest.Response.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.Errorf("Failed to fetch channels in guild %s: %s", serverID, err.Error())
	}

	// Only consider channels the user is managing here so siblings outside
	// this list (e.g. ad-hoc admin channels) don't appear in state.
	managed := make(map[string]struct{}, len(d.Get("channel_ids").([]interface{})))
	for _, raw := range d.Get("channel_ids").([]interface{}) {
		if id, _ := raw.(string); id != "" {
			managed[id] = struct{}{}
		}
	}

	siblings := make([]*discordgo.Channel, 0, len(managed))
	seen := make(map[string]struct{}, len(managed))
	for _, c := range channels {
		if _, ok := managed[c.ID]; !ok {
			continue
		}
		if categoryID == "" {
			if c.ParentID != "" {
				continue
			}
		} else if c.ParentID != categoryID {
			continue
		}
		seen[c.ID] = struct{}{}
		siblings = append(siblings, c)
	}

	for id := range managed {
		if _, ok := seen[id]; !ok {
			tflog.Warn(ctx, "Managed channel not found in guild; it may have been deleted or moved out of scope. The next plan will show a diff.", map[string]interface{}{
				"channel_id":  id,
				"server_id":   serverID,
				"category_id": categoryID,
			})
		}
	}

	sort.SliceStable(siblings, func(i, j int) bool {
		if siblings[i].Position != siblings[j].Position {
			return siblings[i].Position < siblings[j].Position
		}
		return siblings[i].ID < siblings[j].ID
	})

	ordered := make([]interface{}, 0, len(siblings))
	for _, c := range siblings {
		ordered = append(ordered, c.ID)
	}
	d.Set("server_id", serverID)
	if categoryID == "" {
		d.Set("category_id", nil)
	} else {
		d.Set("category_id", categoryID)
	}
	d.Set("channel_ids", ordered)
	return nil
}

func resourceChannelOrderDelete(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	// Removing this resource does not restore any "previous" order — Discord
	// has no such concept. Channels stay wherever they currently are; we just
	// stop managing the ordering.
	d.SetId("")
	return nil
}

func resourceChannelOrderImport(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	serverID, categoryID, err := parseChannelOrderID(d.Id())
	if err != nil {
		return nil, err
	}
	d.Set("server_id", serverID)
	if categoryID == "" {
		d.Set("category_id", nil)
	} else {
		d.Set("category_id", categoryID)
	}
	return []*schema.ResourceData{d}, nil
}

func buildChannelOrderID(serverID, categoryID string) string {
	return serverID + ":" + categoryID
}

func parseChannelOrderID(id string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "", "", fmt.Errorf("expected ID in the form <server_id>:<category_id_or_empty>, got %q", id)
	}
	return parts[0], parts[1], nil
}

func expandChannelOrderIDs(raw []interface{}) ([]string, error) {
	out := make([]string, 0, len(raw))
	for i, v := range raw {
		s, ok := v.(string)
		if !ok || s == "" {
			return nil, fmt.Errorf("channel_ids[%d] is empty; each entry must be a non-empty channel ID", i)
		}
		out = append(out, s)
	}
	return out, nil
}
