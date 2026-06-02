package discord

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/net/context"
)

func resourceDiscordMessage() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMessageCreate,
		ReadContext:   resourceMessageRead,
		UpdateContext: resourceMessageUpdate,
		DeleteContext: resourceMessageDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Description: "A resource to create a message",
		Schema: map[string]*schema.Schema{
			"channel_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the channel the message will be in.",
			},
			"server_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the server this message is in.",
			},
			"author": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the user who wrote the message.",
			},
			"content": {
				AtLeastOneOf: []string{"content", "embed", "file"},
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Text content of message. At least one of `content`, `embed`, or `file` must be set.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return old == strings.TrimSuffix(new, "\r\n")
				},
			},
			"timestamp": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "When the message was sent.",
			},
			"edited_timestamp": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "When the message was edited.",
			},
			"tts": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether this message triggers TTS. (default `false`)",
			},
			"embed": {
				AtLeastOneOf: []string{"content", "embed", "file"},
				Type:         schema.TypeList,
				Optional:     true,
				MaxItems:     1,
				Description:  "An embed block. At least one of `content`, `embed`, or `file` must be set.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"title": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Title of the embed.",
						},
						"description": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Description of the embed.",
						},
						"url": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "URL of the embed.",
						},
						"timestamp": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Timestamp of the embed content.",
						},
						"color": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Color of the embed. Must be an integer color code.",
						},
						"footer": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Footer of the embed.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"text": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Text of the footer.",
									},
									"icon_url": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "URL to an icon to be included in the footer.",
									},
								},
							},
						},
						"image": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Image to be included in the embed.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"url": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "URL of the image to be included in the embed.",
									},
									"proxy_url": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "URL to access the image via Discord's proxy.",
									},
									"height": {
										Type:        schema.TypeInt,
										Optional:    true,
										Description: "Height of the image.",
									},
									"width": {
										Type:        schema.TypeInt,
										Optional:    true,
										Description: "Width of the image.",
									},
								},
							},
						},
						"thumbnail": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Thumbnail to be included in the embed.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"url": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "URL of the thumbnail to be included in the embed.",
									},
									"proxy_url": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "URL to access the thumbnail via Discord's proxy.",
									},
									"height": {
										Type:        schema.TypeInt,
										Optional:    true,
										Description: "Height of the thumbnail.",
									},
									"width": {
										Type:        schema.TypeInt,
										Optional:    true,
										Description: "Width of the thumbnail.",
									},
								},
							},
						},
						"video": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Video to be included in the embed.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"url": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "URL of the video to be included in the embed.",
									},
									"height": {
										Type:        schema.TypeInt,
										Optional:    true,
										Description: "Height of the video.",
									},
									"width": {
										Type:        schema.TypeInt,
										Optional:    true,
										Description: "Width of the video.",
									},
								},
							},
						},
						"provider": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Provider of the embed.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Name of the provider.",
									},
									"url": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "URL of the provider.",
									},
								},
							},
						},
						"author": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Author of the embed.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Name of the author.",
									},
									"url": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "URL of the author.",
									},
									"icon_url": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "URL of the author's icon.",
									},
									"proxy_icon_url": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "URL to access the author's icon via Discord's proxy.",
									},
								},
							},
						},
						"fields": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Fields of the embed.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Name of the field.",
									},
									"value": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Value of the field.",
									},
									"inline": {
										Type:        schema.TypeBool,
										Optional:    true,
										Description: "Whether the field is inline.",
									},
								},
							},
						},
					},
				},
			},
			"pinned": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether this message is pinned. (default `false`)",
			},
			"file": {
				AtLeastOneOf: []string{"content", "embed", "file"},
				Type:         schema.TypeList,
				Optional:     true,
				ForceNew:     true,
				MaxItems:     messageMaxAttachments,
				Description:  fmt.Sprintf("A local file to attach to the message. Up to %d file blocks are supported (Discord's per-message attachment limit). Any change to a `file` block recreates the message — Discord does not allow editing existing attachments in place.", messageMaxAttachments),
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"source": {
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
							Description: "Path to a local file to upload as an attachment.",
						},
						"filename": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							ForceNew:    true,
							Description: "Override the filename Discord stores for the attachment. Defaults to the basename of `source`.",
						},
						"content_type": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							ForceNew:    true,
							Description: "MIME type to send with the upload. Auto-detected from the file extension when omitted.",
						},
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "ID Discord assigned to the attachment.",
						},
						"url": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "CDN URL to the attachment.",
						},
						"proxy_url": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "URL to access the attachment via Discord's proxy.",
						},
						"size": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Size of the attachment in bytes (as reported by Discord).",
						},
					},
				},
			},
			"type": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The type of the message.",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the message.",
			},
		},
	}
}

func resourceMessageCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	channelId := d.Get("channel_id").(string)

	embeds := make([]*discordgo.MessageEmbed, 0, 1)
	if v, ok := d.GetOk("embed"); ok {
		if embed, err := buildEmbed(v.([]interface{})); err != nil {
			return diag.Errorf("Failed to create message in %s: %s", channelId, err.Error())
		} else {
			embeds = append(embeds, embed)
		}
	}

	files, closers, err := buildMessageFiles(d.Get("file").([]interface{}))
	if err != nil {
		return diag.Errorf("Failed to create message in %s: %s", channelId, err.Error())
	}
	defer closeAll(closers)

	message, err := client.ChannelMessageSendComplex(channelId, &discordgo.MessageSend{
		Content: d.Get("content").(string),
		Embeds:  embeds,
		Files:   files,
		TTS:     d.Get("tts").(bool),
	}, discordgo.WithContext(ctx))
	if err != nil {
		return diag.Errorf("Failed to create message in %s: %s", channelId, err.Error())
	}

	d.SetId(message.ID)
	d.Set("type", int(message.Type))
	d.Set("timestamp", message.Timestamp.Format(time.RFC3339))
	d.Set("author", message.Author.ID)
	if len(message.Embeds) > 0 {
		d.Set("embed", unbuildEmbed(message.Embeds[0]))
	} else {
		d.Set("embed", nil)
	}
	d.Set("file", flattenMessageAttachments(d.Get("file").([]interface{}), message.Attachments))
	if message.GuildID != "" {
		d.Set("server_id", message.GuildID)
	}

	if d.Get("pinned").(bool) {
		pinError := client.ChannelMessagePin(channelId, message.ID, discordgo.WithContext(ctx))
		if pinError != nil {
			diags = append(diags, diag.Errorf("Failed to pin message %s in %s: %s", message.ID, channelId, err.Error())...)
		}
	}

	return diags
}

func resourceMessageRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	channelId := d.Get("channel_id").(string)
	messageId := d.Id()
	message, err := client.ChannelMessage(channelId, messageId, discordgo.WithContext(ctx))
	if err != nil {
		return diag.Errorf("Failed to fetch message %s in %s: %s", messageId, channelId, err.Error())
	}

	if message.GuildID != "" {
		d.Set("server_id", message.GuildID)
	}
	d.Set("type", int(message.Type))
	d.Set("tts", message.TTS)
	d.Set("timestamp", message.Timestamp.Format(time.RFC3339))
	d.Set("author", message.Author.ID)
	d.Set("content", message.Content)
	d.Set("pinned", message.Pinned)

	if len(message.Embeds) > 0 {
		d.Set("embed", unbuildEmbed(message.Embeds[0]))
	}
	d.Set("file", flattenMessageAttachments(d.Get("file").([]interface{}), message.Attachments))
	if message.EditedTimestamp != nil {
		d.Set("edited_timestamp", message.EditedTimestamp.Format(time.RFC3339))
	}

	return diags
}

func resourceMessageUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	var diags diag.Diagnostics
	client := m.(*Context).Session

	channelId := d.Get("channel_id").(string)
	messageId := d.Id()

	var content string
	message, err := client.ChannelMessage(channelId, messageId, discordgo.WithContext(ctx))
	if err != nil {
		return diag.Errorf("Failed to fetch message %s in %s: %s", messageId, channelId, err.Error())
	}
	if d.HasChange("content") {
		content = d.Get("content").(string)
	} else {
		content = message.Content
	}

	embeds := make([]*discordgo.MessageEmbed, 0, 1)
	if d.HasChange("embed") {
		var embed *discordgo.MessageEmbed
		_, n := d.GetChange("embed")
		if len(n.([]interface{})) > 0 {
			if e, err := buildEmbed(n.([]interface{})); err != nil {
				return diag.Errorf("Failed to edit message %s in %s: %s", messageId, channelId, err.Error())
			} else {
				embed = e
			}
		}

		embeds = append(embeds, embed)
	}

	editedMessage, err := client.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:      messageId,
		Channel: channelId,
		Content: &content,
		Embeds:  &embeds,
	}, discordgo.WithContext(ctx))
	if err != nil {
		return diag.Errorf("Failed to update message %s in %s: %s", channelId, messageId, err.Error())
	}

	if len(editedMessage.Embeds) > 0 {
		d.Set("embed", unbuildEmbed(message.Embeds[0]))
	} else {
		d.Set("embed", nil)
	}

	d.Set("edited_timestamp", editedMessage.EditedTimestamp.Format(time.RFC3339))

	return diags
}

func resourceMessageDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	channelId := d.Get("channel_id").(string)
	messageId := d.Id()
	if err := client.ChannelMessageDelete(channelId, messageId, discordgo.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to delete message %s in %s: %s", messageId, channelId, err.Error())
	} else {
		return diags
	}
}

// messageMaxAttachments is Discord's hard cap on file attachments per message.
const messageMaxAttachments = 10

// buildMessageFiles opens each configured `file` block for upload. The returned
// closers must be closed by the caller after the request has been sent.
func buildMessageFiles(blocks []interface{}) ([]*discordgo.File, []*os.File, error) {
	if len(blocks) == 0 {
		return nil, nil, nil
	}
	if len(blocks) > messageMaxAttachments {
		return nil, nil, fmt.Errorf("a message may have at most %d file attachments, got %d", messageMaxAttachments, len(blocks))
	}
	files := make([]*discordgo.File, 0, len(blocks))
	closers := make([]*os.File, 0, len(blocks))
	for i, raw := range blocks {
		block, ok := raw.(map[string]interface{})
		if !ok {
			return nil, closers, fmt.Errorf("file block %d is not a map", i)
		}
		source, _ := block["source"].(string)
		if source == "" {
			return nil, closers, fmt.Errorf("file block %d has empty `source`", i)
		}
		fh, err := os.Open(source)
		if err != nil {
			return nil, closers, fmt.Errorf("failed to open %s: %w", source, err)
		}
		closers = append(closers, fh)

		name, _ := block["filename"].(string)
		if name == "" {
			name = filepath.Base(source)
		}
		ctype, _ := block["content_type"].(string)
		if ctype == "" {
			ctype = mime.TypeByExtension(filepath.Ext(name))
		}
		files = append(files, &discordgo.File{
			Name:        name,
			ContentType: ctype,
			Reader:      fh,
		})
	}
	return files, closers, nil
}

func closeAll(files []*os.File) {
	for _, f := range files {
		_ = f.Close()
	}
}

// flattenMessageAttachments merges Discord's response (`attachments`) into the
// list the user configured. Order follows the original config so the per-index
// `source`/`filename`/`content_type` inputs are preserved; computed fields
// (`id`, `url`, `proxy_url`, `size`) come from the API response.
func flattenMessageAttachments(blocks []interface{}, attachments []*discordgo.MessageAttachment) []interface{} {
	if len(blocks) == 0 && len(attachments) == 0 {
		return nil
	}
	out := make([]interface{}, 0, len(blocks))
	for i, raw := range blocks {
		block, _ := raw.(map[string]interface{})
		if block == nil {
			block = map[string]interface{}{}
		}
		merged := map[string]interface{}{
			"source":       block["source"],
			"filename":     block["filename"],
			"content_type": block["content_type"],
		}
		if i < len(attachments) && attachments[i] != nil {
			a := attachments[i]
			merged["id"] = a.ID
			merged["url"] = a.URL
			merged["proxy_url"] = a.ProxyURL
			merged["size"] = a.Size
			if merged["filename"] == "" {
				merged["filename"] = a.Filename
			}
			if merged["content_type"] == "" {
				merged["content_type"] = a.ContentType
			}
		}
		out = append(out, merged)
	}
	return out
}
