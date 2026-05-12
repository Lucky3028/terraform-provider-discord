package discord

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceDiscordRoleConnectionMetadata() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRoleConnectionMetadataCreate,
		ReadContext:   resourceRoleConnectionMetadataRead,
		UpdateContext: resourceRoleConnectionMetadataUpdate,
		DeleteContext: resourceRoleConnectionMetadataDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Description: "A resource to manage application role connection metadata. This defines the criteria that can be used for linked roles in servers where the application is installed.",

		Schema: map[string]*schema.Schema{
			"application_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the application to configure role connection metadata for.",
			},
			"metadata": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    5,
				Description: "List of role connection metadata records. An application can have a maximum of 5.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringLenBetween(1, 50),
							Description:  "Dictionary key for the metadata field (must be a-z, 0-9, or _ characters; 1-50 characters).",
						},
						"name": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringLenBetween(1, 100),
							Description:  "Name of the metadata field (1-100 characters).",
						},
						"type": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntBetween(1, 8),
							Description:  "Type of metadata value. 1=integer_less_than_or_equal, 2=integer_greater_than_or_equal, 3=integer_equal, 4=integer_not_equal, 5=datetime_less_than_or_equal, 6=datetime_greater_than_or_equal, 7=boolean_equal, 8=boolean_not_equal.",
						},
						"description": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringLenBetween(1, 200),
							Description:  "Description of the metadata field (1-200 characters).",
						},
					},
				},
			},
		},
	}
}

func resourceRoleConnectionMetadataCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	appID := d.Get("application_id").(string)
	metadata := buildRoleConnectionMetadata(d)

	if _, err := client.ApplicationRoleConnectionMetadataUpdate(appID, metadata); err != nil {
		return diag.Errorf("Failed to create role connection metadata: %s", err.Error())
	}

	d.SetId(appID)

	return diags
}

func resourceRoleConnectionMetadataRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	appID := d.Id()

	records, err := client.ApplicationRoleConnectionMetadata(appID)
	if err != nil {
		d.SetId("")
		return diags
	}

	d.Set("application_id", appID)

	metadata := make([]map[string]interface{}, len(records))
	for i, r := range records {
		metadata[i] = map[string]interface{}{
			"key":         r.Key,
			"name":        r.Name,
			"type":        int(r.Type),
			"description": r.Description,
		}
	}
	d.Set("metadata", metadata)

	return diags
}

func resourceRoleConnectionMetadataUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	appID := d.Id()
	metadata := buildRoleConnectionMetadata(d)

	if _, err := client.ApplicationRoleConnectionMetadataUpdate(appID, metadata); err != nil {
		return diag.Errorf("Failed to update role connection metadata: %s", err.Error())
	}

	return diags
}

func resourceRoleConnectionMetadataDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Context).Session

	appID := d.Id()

	if _, err := client.ApplicationRoleConnectionMetadataUpdate(appID, []*discordgo.ApplicationRoleConnectionMetadata{}); err != nil {
		return diag.Errorf("Failed to delete role connection metadata: %s", err.Error())
	}

	return diags
}

func buildRoleConnectionMetadata(d *schema.ResourceData) []*discordgo.ApplicationRoleConnectionMetadata {
	metadataList := d.Get("metadata").([]interface{})
	metadata := make([]*discordgo.ApplicationRoleConnectionMetadata, len(metadataList))

	for i, v := range metadataList {
		m := v.(map[string]interface{})
		metadata[i] = &discordgo.ApplicationRoleConnectionMetadata{
			Key:         m["key"].(string),
			Name:        m["name"].(string),
			Type:        discordgo.ApplicationRoleConnectionMetadataType(m["type"].(int)),
			Description: m["description"].(string),
		}
	}

	return metadata
}
