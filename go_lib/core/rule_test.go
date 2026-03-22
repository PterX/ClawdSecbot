package core

import (
	"testing"
)

func TestRuleExpressionValidation(t *testing.T) {
	tests := []struct {
		name    string
		expr    RuleExpression
		wantErr bool
	}{
		{
			name: "valid expression",
			expr: RuleExpression{
				Lang: "json_match",
				Expr: `{"ports": [8080]}`,
			},
			wantErr: false,
		},
		{
			name: "empty language",
			expr: RuleExpression{
				Lang: "",
				Expr: "some expr",
			},
			wantErr: true,
		},
		{
			name: "empty expression",
			expr: RuleExpression{
				Lang: "json_match",
				Expr: "",
			},
			wantErr: true,
		},
		{
			name: "both empty",
			expr: RuleExpression{
				Lang: "",
				Expr: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.expr.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RuleExpression.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAssetFinderRuleValidation(t *testing.T) {
	tests := []struct {
		name    string
		rule    AssetFinderRule
		wantErr bool
	}{
		{
			name: "valid rule",
			rule: AssetFinderRule{
				Code:      "test_code",
				Name:      "Test Rule",
				LifeCycle: RuleLifeCycleRuntime,
				Desc:      "Test description",
				Expression: RuleExpression{
					Lang: "json_match",
					Expr: `{"ports": [8080]}`,
				},
			},
			wantErr: false,
		},
		{
			name: "empty code",
			rule: AssetFinderRule{
				Code:      "",
				Name:      "Test Rule",
				LifeCycle: RuleLifeCycleRuntime,
				Desc:      "Test description",
				Expression: RuleExpression{
					Lang: "json_match",
					Expr: `{"ports": [8080]}`,
				},
			},
			wantErr: true,
		},
		{
			name: "empty name",
			rule: AssetFinderRule{
				Code:      "test_code",
				Name:      "",
				LifeCycle: RuleLifeCycleRuntime,
				Desc:      "Test description",
				Expression: RuleExpression{
					Lang: "json_match",
					Expr: `{"ports": [8080]}`,
				},
			},
			wantErr: true,
		},
		{
			name: "empty description",
			rule: AssetFinderRule{
				Code:      "test_code",
				Name:      "Test Rule",
				LifeCycle: RuleLifeCycleRuntime,
				Desc:      "",
				Expression: RuleExpression{
					Lang: "json_match",
					Expr: `{"ports": [8080]}`,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid lifecycle",
			rule: AssetFinderRule{
				Code:      "test_code",
				Name:      "Test Rule",
				LifeCycle: 999, // Invalid
				Desc:      "Test description",
				Expression: RuleExpression{
					Lang: "json_match",
					Expr: `{"ports": [8080]}`,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid expression",
			rule: AssetFinderRule{
				Code:      "test_code",
				Name:      "Test Rule",
				LifeCycle: RuleLifeCycleRuntime,
				Desc:      "Test description",
				Expression: RuleExpression{
					Lang: "",
					Expr: "",
				},
			},
			wantErr: true,
		},
		{
			name: "static lifecycle",
			rule: AssetFinderRule{
				Code:      "test_code",
				Name:      "Test Rule",
				LifeCycle: RuleLifeCycleStatic,
				Desc:      "Test description",
				Expression: RuleExpression{
					Lang: "json_match",
					Expr: `{"file_paths": ["/etc/config"]}`,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("AssetFinderRule.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewAssetRule(t *testing.T) {
	rule, err := NewAssetRule(
		"test_code",
		"Test Rule",
		RuleLifeCycleRuntime,
		"Test description",
		RuleExpression{
			Lang: "json_match",
			Expr: `{"ports": [8080]}`,
		},
	)

	if err != nil {
		t.Fatalf("NewAssetRule() error = %v", err)
	}

	if rule.Code != "test_code" {
		t.Errorf("Expected Code 'test_code', got '%s'", rule.Code)
	}
	if rule.Name != "Test Rule" {
		t.Errorf("Expected Name 'Test Rule', got '%s'", rule.Name)
	}
	if rule.LifeCycle != RuleLifeCycleRuntime {
		t.Errorf("Expected LifeCycle Runtime, got %v", rule.LifeCycle)
	}
}

func TestNewAssetRuleInvalidExpression(t *testing.T) {
	_, err := NewAssetRule(
		"test_code",
		"Test Rule",
		RuleLifeCycleRuntime,
		"Test description",
		RuleExpression{
			Lang: "",
			Expr: "",
		},
	)

	if err == nil {
		t.Error("NewAssetRule() expected error for invalid expression")
	}
}

func TestAssetFinderRuleWithOS(t *testing.T) {
	rule := AssetFinderRule{
		Code:      "darwin_only",
		Name:      "Darwin Only Rule",
		LifeCycle: RuleLifeCycleRuntime,
		OS:        []string{"darwin"},
		Desc:      "Darwin specific rule",
		Expression: RuleExpression{
			Lang: "json_match",
			Expr: `{"ports": [8080]}`,
		},
	}

	if err := rule.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}

	if len(rule.OS) != 1 || rule.OS[0] != "darwin" {
		t.Errorf("Expected OS ['darwin'], got %v", rule.OS)
	}
}
