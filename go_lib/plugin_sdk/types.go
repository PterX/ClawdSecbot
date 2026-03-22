package plugin_sdk

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// PluginManifest describes a plugin's public identity and capabilities.
type PluginManifest struct {
	PluginID           string   `json:"plugin_id"`
	BotType            string   `json:"bot_type"`
	DisplayName        string   `json:"display_name"`
	APIVersion         string   `json:"api_version"`
	Capabilities       []string `json:"capabilities,omitempty"`
	SupportedPlatforms []string `json:"supported_platforms,omitempty"`
}

// AssetUISchema declares how the host should render an asset card.
// Plugins should provide a stable schema as part of the unified contract.
type AssetUISchema struct {
	ID          string              `json:"id"`
	Version     string              `json:"version,omitempty"`
	TitleRef    string              `json:"title_ref,omitempty"`
	SubtitleRef string              `json:"subtitle_ref,omitempty"`
	Badges      []AssetUIBadge      `json:"badges,omitempty"`
	StatusChips []AssetUIStatusChip `json:"status_chips,omitempty"`
	Sections    []AssetUISection    `json:"sections,omitempty"`
	Actions     []AssetUIAction     `json:"actions,omitempty"`
}

// Clone returns a deep copy of the schema so callers can safely customize it
// per instance without mutating the shared plugin template.
func (s *AssetUISchema) Clone() *AssetUISchema {
	if s == nil {
		return nil
	}

	clone := *s
	if len(s.Badges) > 0 {
		clone.Badges = append([]AssetUIBadge{}, s.Badges...)
	}
	if len(s.StatusChips) > 0 {
		clone.StatusChips = append([]AssetUIStatusChip{}, s.StatusChips...)
	}
	if len(s.Sections) > 0 {
		clone.Sections = make([]AssetUISection, len(s.Sections))
		for i := range s.Sections {
			clone.Sections[i] = s.Sections[i].Clone()
		}
	}
	if len(s.Actions) > 0 {
		clone.Actions = append([]AssetUIAction{}, s.Actions...)
	}
	return &clone
}

type AssetUIBadge struct {
	LabelKey string `json:"label_key,omitempty"`
	Label    string `json:"label,omitempty"`
	ValueRef string `json:"value_ref,omitempty"`
	Tone     string `json:"tone,omitempty"`
}

type AssetUIStatusChip struct {
	LabelKey string `json:"label_key,omitempty"`
	Label    string `json:"label,omitempty"`
	ValueRef string `json:"value_ref,omitempty"`
	Tone     string `json:"tone,omitempty"`
}

type AssetUISection struct {
	Type     string         `json:"type"`
	LabelKey string         `json:"label_key,omitempty"`
	Label    string         `json:"label,omitempty"`
	ValueRef string         `json:"value_ref,omitempty"`
	Items    []AssetUIField `json:"items,omitempty"`
}

// Clone returns a deep copy of the section.
func (s AssetUISection) Clone() AssetUISection {
	clone := s
	if len(s.Items) > 0 {
		clone.Items = append([]AssetUIField{}, s.Items...)
	}
	return clone
}

type AssetUIField struct {
	LabelKey string `json:"label_key,omitempty"`
	Label    string `json:"label,omitempty"`
	ValueRef string `json:"value_ref,omitempty"`
	Tone     string `json:"tone,omitempty"`
}

type AssetUIAction struct {
	Action   string `json:"action"`
	LabelKey string `json:"label_key,omitempty"`
	Label    string `json:"label,omitempty"`
	Variant  string `json:"variant,omitempty"`
}

// BuildInstanceID creates a stable internal identifier from plugin identity
// and normalized instance hints, such as config path or process root.
// The returned value is for routing/persistence only and should not be shown
// as a user-facing label in the UI.
func BuildInstanceID(pluginID string, parts ...string) string {
	normalized := make([]string, 0, len(parts)+1)
	normalized = append(normalized, strings.TrimSpace(strings.ToLower(pluginID)))
	for _, part := range parts {
		part = strings.TrimSpace(strings.ToLower(part))
		if part == "" {
			continue
		}
		normalized = append(normalized, part)
	}

	sum := sha256.Sum256([]byte(strings.Join(normalized, "|")))
	return pluginID + ":" + hex.EncodeToString(sum[:12])
}
