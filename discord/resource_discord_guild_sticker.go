package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/net/context"
)

func resourceDiscordGuildSticker() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGuildStickerCreate,
		ReadContext:   resourceGuildStickerRead,
		UpdateContext: resourceGuildStickerUpdate,
		DeleteContext: resourceGuildStickerDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceGuildStickerImport,
		},

		Description: "A resource to manage a guild sticker.",
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the guild to create the sticker in.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the sticker (2-30 characters).",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Description of the sticker (empty or 2-100 characters).",
			},
			"tags": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Autocomplete/suggestion tags for the sticker (max 200 characters).",
			},
			"file": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Path to the sticker file. Must be PNG, APNG, GIF, or Lottie JSON. Max 512 KB.",
			},
			"format_type": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Format type of the sticker (1=PNG, 2=APNG, 3=Lottie, 4=GIF).",
			},
		},
	}
}

func resourceGuildStickerCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	serverID := d.Get("server_id").(string)
	filePath := d.Get("file").(string)

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return diag.Errorf("Failed to read sticker file %s: %s", filePath, err.Error())
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writer.WriteField("name", d.Get("name").(string))
	writer.WriteField("description", d.Get("description").(string))
	writer.WriteField("tags", d.Get("tags").(string))

	if part, err := createFormFileWithContentType(writer, "file", filepath.Base(filePath)); err != nil {
		return diag.Errorf("Failed to create form file: %s", err.Error())
	} else {
		part.Write(fileData)
	}
	writer.Close()

	endpoint := discordgo.EndpointGuildStickers(serverID)
	if response, err := client.RequestRaw("POST", endpoint, writer.FormDataContentType(), body.Bytes(), endpoint, 0, discordgo.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to create sticker: %s", err.Error())
	} else {
		var sticker discordgo.Sticker
		if err := json.Unmarshal(response, &sticker); err != nil {
			return diag.Errorf("Failed to parse sticker response: %s", err.Error())
		}

		d.SetId(sticker.ID)
		d.Set("format_type", int(sticker.FormatType))
	}

	return diags
}

func resourceGuildStickerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	serverID := d.Get("server_id").(string)
	endpoint := discordgo.EndpointGuildSticker(serverID, d.Id())

	if response, err := client.RequestWithBucketID("GET", endpoint, nil, endpoint, discordgo.WithContext(ctx)); err != nil {
		d.SetId("")
	} else {
		var sticker discordgo.Sticker
		if err := json.Unmarshal(response, &sticker); err != nil {
			return diag.Errorf("Failed to parse sticker response: %s", err.Error())
		}

		d.Set("name", sticker.Name)
		d.Set("description", sticker.Description)
		d.Set("tags", sticker.Tags)
		d.Set("format_type", int(sticker.FormatType))
	}

	return diags
}

func resourceGuildStickerUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	serverID := d.Get("server_id").(string)

	data := struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Tags        string `json:"tags"`
	}{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Tags:        d.Get("tags").(string),
	}

	endpoint := discordgo.EndpointGuildSticker(serverID, d.Id())
	if _, err := client.RequestWithBucketID("PATCH", endpoint, data, endpoint, discordgo.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to update sticker %s: %s", d.Id(), err.Error())
	}

	return diags
}

func resourceGuildStickerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	serverID := d.Get("server_id").(string)
	endpoint := discordgo.EndpointGuildSticker(serverID, d.Id())

	if _, err := client.RequestWithBucketID("DELETE", endpoint, nil, endpoint, discordgo.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to delete sticker %s: %s", d.Id(), err.Error())
	}

	return diags
}

func resourceGuildStickerImport(ctx context.Context, data *schema.ResourceData, i interface{}) ([]*schema.ResourceData, error) {
	if serverId, stickerId, err := parseTwoIds(data.Id()); err != nil {
		return nil, err
	} else {
		data.SetId(stickerId)
		data.Set("server_id", serverId)

		return schema.ImportStatePassthroughContext(ctx, data, i)
	}
}

func createFormFileWithContentType(w *multipart.Writer, fieldname, filename string) (part io.Writer, err error) {
	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldname, filename))
	h.Set("Content-Type", contentType)

	return w.CreatePart(h)
}
