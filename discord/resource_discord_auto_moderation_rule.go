package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// AutoMod trigger types (per Discord API).
const (
	automodTriggerKeyword       = 1 // KEYWORD
	automodTriggerSpam          = 3 // SPAM
	automodTriggerKeywordPreset = 4 // KEYWORD_PRESET
	automodTriggerMentionSpam   = 5 // MENTION_SPAM
	automodTriggerMemberProfile = 6 // MEMBER_PROFILE
)

const (
	automodActionBlockMessage      = 1
	automodActionSendAlertMessage  = 2
	automodActionTimeout           = 3
	automodActionBlockMemberInteract = 4
)

func resourceDiscordAutoModerationRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAutoModRuleCreate,
		ReadContext:   resourceAutoModRuleRead,
		UpdateContext: resourceAutoModRuleUpdate,
		DeleteContext: resourceAutoModRuleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceAutoModRuleImport,
		},
		CustomizeDiff: customizeAutoModRuleDiff,
		Description: "Manages a Discord Auto Moderation rule. " +
			"AutoMod scans messages and member profile updates for violations and runs configured actions (block, alert, timeout). " +
			"Per-guild limits enforced server-side: max 6 KEYWORD rules and one each of SPAM/KEYWORD_PRESET/MENTION_SPAM/MEMBER_PROFILE.",
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the guild this rule applies to.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Display name of the rule.",
			},
			"event_type": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntInSlice([]int{1, 2}),
				Description:  "When the rule fires: `1` = MESSAGE_SEND (message is sent or edited), `2` = MEMBER_UPDATE (member's profile changes).",
			},
			"trigger_type": {
				Type:         schema.TypeInt,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntInSlice([]int{1, 3, 4, 5, 6}),
				Description: "What kind of content triggers the rule. Discord does not allow editing this on an existing rule, so any change forces a replace. " +
					"`1` = KEYWORD, `3` = SPAM, `4` = KEYWORD_PRESET, `5` = MENTION_SPAM, `6` = MEMBER_PROFILE.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the rule is active.",
			},
			"exempt_roles": {
				Type:        schema.TypeSet,
				Optional:    true,
				MaxItems:    20,
				Description: "Role IDs whose members bypass this rule.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"exempt_channels": {
				Type:        schema.TypeSet,
				Optional:    true,
				MaxItems:    50,
				Description: "Channel IDs where this rule is not enforced.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"creator_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the user who created the rule (set by Discord at creation).",
			},
			"trigger_metadata": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Configuration specific to the chosen `trigger_type`. Required for trigger types 1, 4, 5, and 6.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"keyword_filter": {
							Type:        schema.TypeSet,
							Optional:    true,
							Description: "Substrings that trigger the rule. Used by KEYWORD (1) and MEMBER_PROFILE (6).",
							MaxItems:    1000,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"regex_patterns": {
							Type:        schema.TypeSet,
							Optional:    true,
							Description: "Rust-flavor regex patterns that trigger the rule. Used by KEYWORD (1) and MEMBER_PROFILE (6).",
							MaxItems:    10,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"presets": {
							Type:        schema.TypeSet,
							Optional:    true,
							Description: "Discord-built preset categories: `1` = profanity, `2` = sexual content, `3` = slurs. Only valid for KEYWORD_PRESET (4).",
							MaxItems:    3,
							Elem: &schema.Schema{
								Type:         schema.TypeInt,
								ValidateFunc: validation.IntInSlice([]int{1, 2, 3}),
							},
						},
						"allow_list": {
							Type:        schema.TypeSet,
							Optional:    true,
							Description: "Substrings exempted from triggering the rule. Used by KEYWORD (1), KEYWORD_PRESET (4), and MEMBER_PROFILE (6).",
							MaxItems:    1000,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"mention_total_limit": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(0, 50),
							Description:  "Maximum number of unique role and user mentions allowed per message. Only valid for MENTION_SPAM (5).",
						},
						"mention_raid_protection_enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Whether to detect mention-raid behaviour. Only valid for MENTION_SPAM (5).",
						},
					},
				},
			},
			"actions": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "Actions to run when the rule is triggered. At least one action is required.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntInSlice([]int{1, 2, 3, 4}),
							Description:  "`1` = BLOCK_MESSAGE, `2` = SEND_ALERT_MESSAGE, `3` = TIMEOUT (requires `MODERATE_MEMBERS`), `4` = BLOCK_MEMBER_INTERACTION (MEMBER_PROFILE rules only).",
						},
						"metadata": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Action-specific metadata. Required for types 2 (channel_id) and 3 (duration_seconds); optional for type 1 (custom_message); ignored for type 4.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"channel_id": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "Channel to send alerts to. Required when action `type = 2` (SEND_ALERT_MESSAGE).",
									},
									"duration_seconds": {
										Type:         schema.TypeInt,
										Optional:     true,
										ValidateFunc: validation.IntBetween(1, 2419200),
										Description:  "Timeout duration in seconds (max 2419200 = 28 days). Required when action `type = 3` (TIMEOUT).",
									},
									"custom_message": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validation.StringLenBetween(0, 150),
										Description:  "Custom block message shown to the user. Only meaningful when action `type = 1` (BLOCK_MESSAGE).",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// ---------- Wire types ----------

type automodRule struct {
	ID              string            `json:"id,omitempty"`
	GuildID         string            `json:"guild_id,omitempty"`
	Name            string            `json:"name"`
	CreatorID       string            `json:"creator_id,omitempty"`
	EventType       int               `json:"event_type"`
	TriggerType     int               `json:"trigger_type,omitempty"`
	TriggerMetadata *automodTriggerMD `json:"trigger_metadata,omitempty"`
	Actions         []automodAction   `json:"actions"`
	Enabled         bool              `json:"enabled"`
	ExemptRoles     []string          `json:"exempt_roles"`
	ExemptChannels  []string          `json:"exempt_channels"`
}

type automodTriggerMD struct {
	KeywordFilter                []string `json:"keyword_filter,omitempty"`
	RegexPatterns                []string `json:"regex_patterns,omitempty"`
	Presets                      []int    `json:"presets,omitempty"`
	AllowList                    []string `json:"allow_list,omitempty"`
	MentionTotalLimit            *int     `json:"mention_total_limit,omitempty"`
	MentionRaidProtectionEnabled *bool    `json:"mention_raid_protection_enabled,omitempty"`
}

type automodAction struct {
	Type     int                `json:"type"`
	Metadata *automodActionMeta `json:"metadata,omitempty"`
}

type automodActionMeta struct {
	ChannelID       string `json:"channel_id,omitempty"`
	DurationSeconds *int   `json:"duration_seconds,omitempty"`
	CustomMessage   string `json:"custom_message,omitempty"`
}

// ---------- CRUD ----------

func resourceAutoModRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Context).Session
	serverID := d.Get("server_id").(string)
	rule := buildAutoModRuleFromState(d)

	endpoint := discordgo.EndpointGuildAutoModerationRules(serverID)
	body, err := client.RequestWithBucketID("POST", endpoint, rule, endpoint, discordgo.WithContext(ctx))
	if err != nil {
		return diag.Errorf("Failed to create AutoMod rule in guild %s: %s", serverID, err.Error())
	}
	var created automodRule
	if err := json.Unmarshal(body, &created); err != nil {
		return diag.Errorf("Failed to parse AutoMod rule response: %s", err.Error())
	}
	d.SetId(created.ID)
	return flattenAutoModRule(d, &created)
}

func resourceAutoModRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Context).Session
	serverID := d.Get("server_id").(string)
	endpoint := discordgo.EndpointGuildAutoModerationRule(serverID, d.Id())
	body, err := client.RequestWithBucketID("GET", endpoint, nil, endpoint, discordgo.WithContext(ctx))
	if err != nil {
		if rest, ok := err.(*discordgo.RESTError); ok && rest.Response != nil && rest.Response.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.Errorf("Failed to fetch AutoMod rule %s: %s", d.Id(), err.Error())
	}
	var rule automodRule
	if err := json.Unmarshal(body, &rule); err != nil {
		return diag.Errorf("Failed to parse AutoMod rule response: %s", err.Error())
	}
	return flattenAutoModRule(d, &rule)
}

func resourceAutoModRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Context).Session
	serverID := d.Get("server_id").(string)
	rule := buildAutoModRuleFromState(d)
	// Discord forbids changing trigger_type via PATCH; the field is ForceNew
	// so Terraform handles replacement, but belt-and-braces: don't send it.
	rule.TriggerType = 0

	endpoint := discordgo.EndpointGuildAutoModerationRule(serverID, d.Id())
	if _, err := client.RequestWithBucketID("PATCH", endpoint, rule, endpoint, discordgo.WithContext(ctx)); err != nil {
		return diag.Errorf("Failed to update AutoMod rule %s: %s", d.Id(), err.Error())
	}
	return resourceAutoModRuleRead(ctx, d, m)
}

func resourceAutoModRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Context).Session
	serverID := d.Get("server_id").(string)
	endpoint := discordgo.EndpointGuildAutoModerationRule(serverID, d.Id())
	if _, err := client.RequestWithBucketID("DELETE", endpoint, nil, endpoint, discordgo.WithContext(ctx)); err != nil {
		if rest, ok := err.(*discordgo.RESTError); ok && rest.Response != nil && rest.Response.StatusCode == 404 {
			return nil
		}
		return diag.Errorf("Failed to delete AutoMod rule %s: %s", d.Id(), err.Error())
	}
	return nil
}

func resourceAutoModRuleImport(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("import id must be in the form <server_id>:<rule_id>, got %q", d.Id())
	}
	d.Set("server_id", parts[0])
	d.SetId(parts[1])
	return []*schema.ResourceData{d}, nil
}

// ---------- Builders / flatteners ----------

func buildAutoModRuleFromState(d *schema.ResourceData) *automodRule {
	rule := &automodRule{
		Name:           d.Get("name").(string),
		EventType:      d.Get("event_type").(int),
		TriggerType:    d.Get("trigger_type").(int),
		Enabled:        d.Get("enabled").(bool),
		ExemptRoles:    expandStringSetSlice(d.Get("exempt_roles")),
		ExemptChannels: expandStringSetSlice(d.Get("exempt_channels")),
		Actions:        expandAutoModActions(d.Get("actions").([]interface{})),
	}
	if md := buildAutoModTriggerMetadata(d); md != nil {
		rule.TriggerMetadata = md
	}
	return rule
}

func buildAutoModTriggerMetadata(d *schema.ResourceData) *automodTriggerMD {
	raw, ok := d.GetOk("trigger_metadata")
	if !ok {
		return nil
	}
	list := raw.([]interface{})
	if len(list) == 0 || list[0] == nil {
		return nil
	}
	block := list[0].(map[string]interface{})
	md := &automodTriggerMD{
		KeywordFilter: expandStringSetSlice(block["keyword_filter"]),
		RegexPatterns: expandStringSetSlice(block["regex_patterns"]),
		AllowList:     expandStringSetSlice(block["allow_list"]),
		Presets:       expandIntSetSlice(block["presets"]),
	}
	if v, ok := block["mention_total_limit"]; ok {
		if i, ok := v.(int); ok && i > 0 {
			md.MentionTotalLimit = &i
		}
	}
	if v, ok := block["mention_raid_protection_enabled"]; ok {
		if b, ok := v.(bool); ok && b {
			md.MentionRaidProtectionEnabled = &b
		}
	}
	return md
}

func expandAutoModActions(list []interface{}) []automodAction {
	out := make([]automodAction, 0, len(list))
	for _, raw := range list {
		block := raw.(map[string]interface{})
		action := automodAction{Type: block["type"].(int)}
		if mdList, _ := block["metadata"].([]interface{}); len(mdList) > 0 && mdList[0] != nil {
			mdBlock := mdList[0].(map[string]interface{})
			meta := &automodActionMeta{}
			if s, _ := mdBlock["channel_id"].(string); s != "" {
				meta.ChannelID = s
			}
			if s, _ := mdBlock["custom_message"].(string); s != "" {
				meta.CustomMessage = s
			}
			if i, _ := mdBlock["duration_seconds"].(int); i > 0 {
				meta.DurationSeconds = &i
			}
			if meta.ChannelID != "" || meta.CustomMessage != "" || meta.DurationSeconds != nil {
				action.Metadata = meta
			}
		}
		out = append(out, action)
	}
	return out
}

func flattenAutoModRule(d *schema.ResourceData, rule *automodRule) diag.Diagnostics {
	d.Set("server_id", rule.GuildID)
	d.Set("name", rule.Name)
	d.Set("event_type", rule.EventType)
	d.Set("trigger_type", rule.TriggerType)
	d.Set("enabled", rule.Enabled)
	d.Set("creator_id", rule.CreatorID)
	d.Set("exempt_roles", rule.ExemptRoles)
	d.Set("exempt_channels", rule.ExemptChannels)
	d.Set("actions", flattenAutoModActions(rule.Actions))
	d.Set("trigger_metadata", flattenAutoModTriggerMetadata(rule.TriggerMetadata))
	return nil
}

func flattenAutoModActions(actions []automodAction) []interface{} {
	out := make([]interface{}, 0, len(actions))
	for _, a := range actions {
		entry := map[string]interface{}{"type": a.Type}
		if a.Metadata != nil {
			md := map[string]interface{}{}
			if a.Metadata.ChannelID != "" {
				md["channel_id"] = a.Metadata.ChannelID
			}
			if a.Metadata.CustomMessage != "" {
				md["custom_message"] = a.Metadata.CustomMessage
			}
			if a.Metadata.DurationSeconds != nil {
				md["duration_seconds"] = *a.Metadata.DurationSeconds
			}
			if len(md) > 0 {
				entry["metadata"] = []interface{}{md}
			}
		}
		out = append(out, entry)
	}
	return out
}

func flattenAutoModTriggerMetadata(md *automodTriggerMD) []interface{} {
	if md == nil {
		return nil
	}
	if len(md.KeywordFilter) == 0 && len(md.RegexPatterns) == 0 && len(md.Presets) == 0 &&
		len(md.AllowList) == 0 && md.MentionTotalLimit == nil && md.MentionRaidProtectionEnabled == nil {
		return nil
	}
	entry := map[string]interface{}{}
	if len(md.KeywordFilter) > 0 {
		entry["keyword_filter"] = md.KeywordFilter
	}
	if len(md.RegexPatterns) > 0 {
		entry["regex_patterns"] = md.RegexPatterns
	}
	if len(md.Presets) > 0 {
		entry["presets"] = md.Presets
	}
	if len(md.AllowList) > 0 {
		entry["allow_list"] = md.AllowList
	}
	if md.MentionTotalLimit != nil {
		entry["mention_total_limit"] = *md.MentionTotalLimit
	}
	if md.MentionRaidProtectionEnabled != nil {
		entry["mention_raid_protection_enabled"] = *md.MentionRaidProtectionEnabled
	}
	return []interface{}{entry}
}

// ---------- CustomizeDiff ----------

func customizeAutoModRuleDiff(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	triggerType, _ := d.Get("trigger_type").(int)

	mdList, _ := d.Get("trigger_metadata").([]interface{})
	var md map[string]interface{}
	if len(mdList) > 0 && mdList[0] != nil {
		md = mdList[0].(map[string]interface{})
	}

	keywordFilterSet := setHasItems(md, "keyword_filter")
	regexPatternsSet := setHasItems(md, "regex_patterns")
	presetsSet := setHasItems(md, "presets")
	allowListSet := setHasItems(md, "allow_list")
	mentionTotalLimit := intFromMap(md, "mention_total_limit")
	mentionRaidProtection := boolFromMap(md, "mention_raid_protection_enabled")

	switch triggerType {
	case automodTriggerKeyword, automodTriggerMemberProfile:
		if !keywordFilterSet && !regexPatternsSet {
			return fmt.Errorf("trigger_type %d requires at least one of trigger_metadata.keyword_filter or trigger_metadata.regex_patterns", triggerType)
		}
		if presetsSet {
			return fmt.Errorf("trigger_metadata.presets is only valid for trigger_type 4 (KEYWORD_PRESET)")
		}
		if mentionTotalLimit > 0 || mentionRaidProtection {
			return fmt.Errorf("trigger_metadata.mention_* fields are only valid for trigger_type 5 (MENTION_SPAM)")
		}
	case automodTriggerSpam:
		if keywordFilterSet || regexPatternsSet || presetsSet || allowListSet || mentionTotalLimit > 0 || mentionRaidProtection {
			return fmt.Errorf("trigger_type 3 (SPAM) does not accept any trigger_metadata fields")
		}
	case automodTriggerKeywordPreset:
		if !presetsSet {
			return fmt.Errorf("trigger_type 4 (KEYWORD_PRESET) requires trigger_metadata.presets")
		}
		if keywordFilterSet || regexPatternsSet || mentionTotalLimit > 0 || mentionRaidProtection {
			return fmt.Errorf("trigger_type 4 (KEYWORD_PRESET) only accepts trigger_metadata.presets and trigger_metadata.allow_list")
		}
	case automodTriggerMentionSpam:
		if mentionTotalLimit == 0 && !mentionRaidProtection {
			return fmt.Errorf("trigger_type 5 (MENTION_SPAM) requires at least one of trigger_metadata.mention_total_limit or trigger_metadata.mention_raid_protection_enabled")
		}
		if keywordFilterSet || regexPatternsSet || presetsSet || allowListSet {
			return fmt.Errorf("trigger_type 5 (MENTION_SPAM) only accepts trigger_metadata.mention_* fields")
		}
	}

	actions, _ := d.Get("actions").([]interface{})
	for i, raw := range actions {
		block := raw.(map[string]interface{})
		actionType := block["type"].(int)
		var meta map[string]interface{}
		if mdList, _ := block["metadata"].([]interface{}); len(mdList) > 0 && mdList[0] != nil {
			meta = mdList[0].(map[string]interface{})
		}
		channelID, _ := meta["channel_id"].(string)
		customMessage, _ := meta["custom_message"].(string)
		duration, _ := meta["duration_seconds"].(int)

		switch actionType {
		case automodActionBlockMessage:
			if channelID != "" || duration > 0 {
				return fmt.Errorf("actions[%d]: type 1 (BLOCK_MESSAGE) only accepts metadata.custom_message", i)
			}
		case automodActionSendAlertMessage:
			if channelID == "" {
				return fmt.Errorf("actions[%d]: type 2 (SEND_ALERT_MESSAGE) requires metadata.channel_id", i)
			}
			if customMessage != "" || duration > 0 {
				return fmt.Errorf("actions[%d]: type 2 (SEND_ALERT_MESSAGE) only accepts metadata.channel_id", i)
			}
		case automodActionTimeout:
			if duration == 0 {
				return fmt.Errorf("actions[%d]: type 3 (TIMEOUT) requires metadata.duration_seconds", i)
			}
			if channelID != "" || customMessage != "" {
				return fmt.Errorf("actions[%d]: type 3 (TIMEOUT) only accepts metadata.duration_seconds", i)
			}
			if triggerType != automodTriggerKeyword && triggerType != automodTriggerKeywordPreset && triggerType != automodTriggerMentionSpam {
				return fmt.Errorf("actions[%d]: type 3 (TIMEOUT) is only allowed for trigger_type 1, 4, or 5", i)
			}
		case automodActionBlockMemberInteract:
			if channelID != "" || customMessage != "" || duration > 0 {
				return fmt.Errorf("actions[%d]: type 4 (BLOCK_MEMBER_INTERACTION) does not accept any metadata fields", i)
			}
			if triggerType != automodTriggerMemberProfile {
				return fmt.Errorf("actions[%d]: type 4 (BLOCK_MEMBER_INTERACTION) is only allowed for trigger_type 6 (MEMBER_PROFILE)", i)
			}
		}
	}
	return nil
}

// ---------- Helpers ----------

func expandStringSetSlice(v interface{}) []string {
	if v == nil {
		return []string{}
	}
	set, ok := v.(*schema.Set)
	if !ok {
		return []string{}
	}
	items := set.List()
	out := make([]string, 0, len(items))
	for _, x := range items {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func expandIntSetSlice(v interface{}) []int {
	if v == nil {
		return nil
	}
	set, ok := v.(*schema.Set)
	if !ok {
		return nil
	}
	items := set.List()
	out := make([]int, 0, len(items))
	for _, x := range items {
		if i, ok := x.(int); ok {
			out = append(out, i)
		}
	}
	return out
}

func setHasItems(m map[string]interface{}, key string) bool {
	if m == nil {
		return false
	}
	if set, ok := m[key].(*schema.Set); ok {
		return set.Len() > 0
	}
	return false
}

func intFromMap(m map[string]interface{}, key string) int {
	if m == nil {
		return 0
	}
	if v, ok := m[key].(int); ok {
		return v
	}
	return 0
}

func boolFromMap(m map[string]interface{}, key string) bool {
	if m == nil {
		return false
	}
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
