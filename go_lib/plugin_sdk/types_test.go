package plugin_sdk

import (
	"strings"
	"testing"
)

func TestAssetUISchemaClone(t *testing.T) {
	schema := &AssetUISchema{
		ID:      "schema.v1",
		Version: "1",
		Badges: []AssetUIBadge{
			{LabelKey: "badge.one", ValueRef: "metadata.one"},
		},
		StatusChips: []AssetUIStatusChip{
			{LabelKey: "chip.one", ValueRef: "metadata.one"},
		},
		Sections: []AssetUISection{
			{
				Type: "kv_list",
				Items: []AssetUIField{
					{LabelKey: "field.one", ValueRef: "metadata.one"},
				},
			},
		},
		Actions: []AssetUIAction{
			{Action: "open_config"},
		},
	}

	clone := schema.Clone()
	if clone == nil {
		t.Fatal("Expected clone to be created")
	}
	if clone == schema {
		t.Fatal("Expected clone to be a distinct pointer")
	}

	clone.Badges[0].Label = "changed"
	clone.Sections[0].Items[0].Label = "changed"
	clone.Actions[0].Label = "changed"

	if schema.Badges[0].Label != "" {
		t.Fatal("Expected original badges to remain unchanged")
	}
	if schema.Sections[0].Items[0].Label != "" {
		t.Fatal("Expected original section items to remain unchanged")
	}
	if schema.Actions[0].Label != "" {
		t.Fatal("Expected original actions to remain unchanged")
	}
}

func TestAssetUISchemaCloneNil(t *testing.T) {
	var schema *AssetUISchema
	if schema.Clone() != nil {
		t.Fatal("Expected nil schema clone to remain nil")
	}
}

func TestBuildInstanceIDNormalizesAndHashesInstanceHints(t *testing.T) {
	instanceA := BuildInstanceID(
		"com.example.plugin",
		" C:\\Bots\\Alpha\\config.json ",
		"C:\\Bots\\Alpha",
	)
	instanceB := BuildInstanceID(
		"com.example.plugin",
		"c:\\bots\\alpha\\config.json",
		"c:\\bots\\alpha",
	)
	instanceC := BuildInstanceID(
		"com.example.plugin",
		"c:\\bots\\beta\\config.json",
		"c:\\bots\\beta",
	)

	if instanceA != instanceB {
		t.Fatalf("Expected normalized instance IDs to match, got %q and %q", instanceA, instanceB)
	}
	if instanceA == instanceC {
		t.Fatalf("Expected different instance hints to produce different IDs, got %q", instanceA)
	}
	if !strings.HasPrefix(instanceA, "com.example.plugin:") {
		t.Fatalf("Expected plugin prefix in instance ID, got %q", instanceA)
	}
	if strings.Contains(strings.ToLower(instanceA), "alpha") || strings.Contains(strings.ToLower(instanceA), "config.json") {
		t.Fatalf("Expected instance ID to hide raw instance details, got %q", instanceA)
	}
}
