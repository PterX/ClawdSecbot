package core

import (
	"encoding/json"
	"testing"
)

func TestRiskJSONSerialization(t *testing.T) {
	risk := Risk{
		ID:          "test_risk",
		Title:       "Test Risk",
		Description: "This is a test risk",
		Level:       RiskLevelHigh,
		Args: map[string]interface{}{
			"port": 8080,
			"path": "/tmp/test",
		},
		Mitigation: &Mitigation{
			Type: "form",
			FormSchema: []FormItem{
				{Key: "action", Label: "Action", Type: "select", Options: []string{"block", "allow"}},
			},
		},
	}

	// Test marshaling
	data, err := json.Marshal(risk)
	if err != nil {
		t.Fatalf("Failed to marshal risk: %v", err)
	}

	// Test unmarshaling
	var unmarshaled Risk
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal risk: %v", err)
	}

	if unmarshaled.ID != risk.ID {
		t.Errorf("Expected ID %s, got %s", risk.ID, unmarshaled.ID)
	}
	if unmarshaled.Level != risk.Level {
		t.Errorf("Expected Level %s, got %s", risk.Level, unmarshaled.Level)
	}
	if unmarshaled.Mitigation.Type != "form" {
		t.Errorf("Expected Mitigation.Type 'form', got %s", unmarshaled.Mitigation.Type)
	}
	if len(unmarshaled.Mitigation.FormSchema) != 1 {
		t.Errorf("Expected 1 form schema, got %d", len(unmarshaled.Mitigation.FormSchema))
	}
}

func TestMitigationSuggestionJSON(t *testing.T) {
	mitigation := Mitigation{
		Type:        "suggestion",
		Title:       "Security Recommendations",
		Description: "Consider the following actions",
		Suggestions: []SuggestionGroup{
			{
				Priority: "P0",
				Category: "Network",
				Items: []SuggestionItem{
					{Action: "block", Detail: "Block suspicious IP", Command: "iptables -A INPUT -s 1.2.3.4 -j DROP"},
				},
			},
		},
	}

	data, err := json.Marshal(mitigation)
	if err != nil {
		t.Fatalf("Failed to marshal mitigation: %v", err)
	}

	var unmarshaled Mitigation
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal mitigation: %v", err)
	}

	if unmarshaled.Type != "suggestion" {
		t.Errorf("Expected type 'suggestion', got %s", unmarshaled.Type)
	}
	if len(unmarshaled.Suggestions) != 1 {
		t.Errorf("Expected 1 suggestion group, got %d", len(unmarshaled.Suggestions))
	}
	if unmarshaled.Suggestions[0].Items[0].Command == "" {
		t.Error("Expected command to be preserved")
	}
}

func TestFormItemValidation(t *testing.T) {
	tests := []struct {
		name     string
		item     FormItem
		expected string
	}{
		{
			name: "text field",
			item: FormItem{
				Key:   "username",
				Label: "Username",
				Type:  "text",
			},
			expected: "text",
		},
		{
			name: "select field",
			item: FormItem{
				Key:     "action",
				Label:   "Action",
				Type:    "select",
				Options: []string{"allow", "block"},
			},
			expected: "select",
		},
		{
			name: "password field",
			item: FormItem{
				Key:  "api_key",
				Label: "API Key",
				Type: "password",
			},
			expected: "password",
		},
		{
			name: "boolean field",
			item: FormItem{
				Key:  "enabled",
				Label: "Enabled",
				Type: "boolean",
			},
			expected: "boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.item)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}
			var item FormItem
			if err := json.Unmarshal(data, &item); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}
			if item.Type != tt.expected {
				t.Errorf("Expected type %s, got %s", tt.expected, item.Type)
			}
		})
	}
}

func TestRiskLevelConstants(t *testing.T) {
	if RiskLevelLow != "low" {
		t.Error("RiskLevelLow should be 'low'")
	}
	if RiskLevelMedium != "medium" {
		t.Error("RiskLevelMedium should be 'medium'")
	}
	if RiskLevelHigh != "high" {
		t.Error("RiskLevelHigh should be 'high'")
	}
	if RiskLevelCritical != "critical" {
		t.Error("RiskLevelCritical should be 'critical'")
	}
}
