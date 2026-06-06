package discord

import (
	"encoding/json"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/net/context"
)

func resourceDiscordThread() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceThreadCreate,
		ReadContext:   resourceThreadRead,
		UpdateContext: resourceThreadUpdate,
		DeleteContext: resourceThreadDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "A resource to create a thread (forum post) in a forum channel.",
		Schema: map[string]*schema.Schema{
			"channel_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the forum channel this thread belongs to.",
			},
			"server_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the server the thread is in.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Title of the thread (1-100 characters).",
				ValidateFunc: func(val interface{}, key string) (warns []string, errors []error) {
					v := val.(string)
					if len(v) < 1 || len(v) > 100 {
						errors = append(errors, fmt.Errorf("%s must be between 1 and 100 characters, got %d", key, len(v)))
					}
					return
				},
			},
			"message": {
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				MaxItems:    1,
				Description: "First message posted in the thread. Changing it recreates the thread because Discord exposes no edit-first-message API for forum threads.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"content": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Text content of the first message.",
						},
					},
				},
			},
			"applied_tags": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "IDs of tags (from the parent forum's `available_tags`) to apply to this thread.",
				MaxItems:    5,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"auto_archive_duration": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1440,
				Description: "Minutes of inactivity before the thread is automatically archived. Discord accepts 60, 1440, 4320, or 10080.",
				ValidateFunc: func(val interface{}, key string) (warns []string, errors []error) {
					v := val.(int)
					switch v {
					case 60, 1440, 4320, 10080:
					default:
						errors = append(errors, fmt.Errorf("%s must be one of 60, 1440, 4320, 10080; got %d", key, v))
					}
					return
				},
			},
			"rate_limit_per_user": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "Slowmode for the thread in seconds (0-21600).",
				ValidateFunc: func(val interface{}, key string) (warns []string, errors []error) {
					v := val.(int)
					if v < 0 || v > 21600 {
						errors = append(errors, fmt.Errorf("%s must be 0-21600 seconds, got %d", key, v))
					}
					return
				},
			},
			"archived": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the thread is currently archived (Discord auto-archives after `auto_archive_duration` of inactivity).",
			},
			"locked": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the thread is locked (only moderators can unarchive).",
			},
		},
	}
}

type forumThreadCreate struct {
	Name                string                `json:"name"`
	AutoArchiveDuration int                   `json:"auto_archive_duration,omitempty"`
	RateLimitPerUser    int                   `json:"rate_limit_per_user,omitempty"`
	AppliedTags         []string              `json:"applied_tags,omitempty"`
	Message             forumThreadMessage    `json:"message"`
}

type forumThreadMessage struct {
	Content string `json:"content"`
}

type threadPatch struct {
	Name                *string  `json:"name,omitempty"`
	AutoArchiveDuration *int     `json:"auto_archive_duration,omitempty"`
	RateLimitPerUser    *int     `json:"rate_limit_per_user,omitempty"`
	AppliedTags         []string `json:"applied_tags,omitempty"`
}

type threadReadResponse struct {
	ID               string                 `json:"id"`
	GuildID          string                 `json:"guild_id"`
	ParentID         string                 `json:"parent_id"`
	Name             string                 `json:"name"`
	AppliedTags      []string               `json:"applied_tags"`
	RateLimitPerUser int                    `json:"rate_limit_per_user"`
	ThreadMetadata   *threadMetadataPayload `json:"thread_metadata"`
}

type threadMessagePayload struct {
	Content string `json:"content"`
}

type threadMetadataPayload struct {
	Archived            bool `json:"archived"`
	Locked              bool `json:"locked"`
	AutoArchiveDuration int  `json:"auto_archive_duration"`
}

func resourceThreadCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Context).Session
	channelID := d.Get("channel_id").(string)

	messageBlocks := d.Get("message").([]interface{})
	if len(messageBlocks) == 0 {
		return diag.Errorf("message block is required")
	}
	messageBlock := messageBlocks[0].(map[string]interface{})

	payload := forumThreadCreate{
		Name:                d.Get("name").(string),
		AutoArchiveDuration: d.Get("auto_archive_duration").(int),
		RateLimitPerUser:    d.Get("rate_limit_per_user").(int),
		AppliedTags:         expandStringSet(d.Get("applied_tags")),
		Message: forumThreadMessage{
			Content: messageBlock["content"].(string),
		},
	}

	endpoint := discordgo.EndpointChannelThreads(channelID)
	body, err := client.RequestWithBucketID("POST", endpoint, payload, endpoint, discordgo.WithContext(ctx))
	if err != nil {
		return diag.Errorf("Failed to create thread in channel %s: %s", channelID, err.Error())
	}

	var resp threadReadResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return diag.Errorf("Failed to parse thread response: %s", err.Error())
	}

	d.SetId(resp.ID)
	return flattenThreadIntoState(d, &resp)
}

func resourceThreadRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Context).Session

	endpoint := discordgo.EndpointChannel(d.Id())
	body, err := client.RequestWithBucketID("GET", endpoint, nil, endpoint, discordgo.WithContext(ctx))
	if err != nil {
		if rest, ok := err.(*discordgo.RESTError); ok && rest.Response != nil && rest.Response.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.Errorf("Failed to fetch thread %s: %s", d.Id(), err.Error())
	}

	var resp threadReadResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return diag.Errorf("Failed to parse thread response: %s", err.Error())
	}

	if diags := flattenThreadIntoState(d, &resp); diags.HasError() {
		return diags
	}

	// Fetch the first (starter) message so the `message` block reflects
	// Discord state. For forum threads the starter message ID equals the
	// thread ID. Required for `terraform import` to produce no spurious
	// "forces replacement" diff on the `message` block.
	if _, alreadySet := d.GetOk("message"); !alreadySet {
		msgEndpoint := discordgo.EndpointChannelMessage(d.Id(), d.Id())
		msgBody, msgErr := client.RequestWithBucketID("GET", msgEndpoint, nil, msgEndpoint, discordgo.WithContext(ctx))
		if msgErr == nil {
			var msg threadMessagePayload
			if jsonErr := json.Unmarshal(msgBody, &msg); jsonErr == nil {
				d.Set("message", []interface{}{
					map[string]interface{}{"content": msg.Content},
				})
			}
		}
	}

	return nil
}

func resourceThreadUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Context).Session

	patch := threadPatch{}
	if d.HasChange("name") {
		v := d.Get("name").(string)
		patch.Name = &v
	}
	if d.HasChange("auto_archive_duration") {
		v := d.Get("auto_archive_duration").(int)
		patch.AutoArchiveDuration = &v
	}
	if d.HasChange("rate_limit_per_user") {
		v := d.Get("rate_limit_per_user").(int)
		patch.RateLimitPerUser = &v
	}
	if d.HasChange("applied_tags") {
		patch.AppliedTags = expandStringSet(d.Get("applied_tags"))
	}

	endpoint := discordgo.EndpointChannel(d.Id())
	if _, err := client.RequestWithBucketID("PATCH", endpoint, patch, endpoint, discordgo.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to update thread %s: %s", d.Id(), err.Error())
	}
	return resourceThreadRead(ctx, d, m)
}

func resourceThreadDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Context).Session
	if _, err := client.ChannelDelete(d.Id(), discordgo.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to delete thread %s: %s", d.Id(), err.Error())
	}
	return nil
}

func flattenThreadIntoState(d *schema.ResourceData, resp *threadReadResponse) diag.Diagnostics {
	d.Set("server_id", resp.GuildID)
	d.Set("channel_id", resp.ParentID)
	d.Set("name", resp.Name)
	d.Set("rate_limit_per_user", resp.RateLimitPerUser)
	if resp.AppliedTags != nil {
		d.Set("applied_tags", resp.AppliedTags)
	}
	if resp.ThreadMetadata != nil {
		d.Set("archived", resp.ThreadMetadata.Archived)
		d.Set("locked", resp.ThreadMetadata.Locked)
		d.Set("auto_archive_duration", resp.ThreadMetadata.AutoArchiveDuration)
	}
	return nil
}

func expandStringSet(v interface{}) []string {
	if v == nil {
		return nil
	}
	set, ok := v.(*schema.Set)
	if !ok {
		return nil
	}
	list := set.List()
	out := make([]string, 0, len(list))
	for _, x := range list {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
