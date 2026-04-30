package modelfactory

import (
	"encoding/json"
	"strings"
	"testing"

	"go_lib/core/repository"
)

func TestTestModelConnectionInternalReturnsEnvelopeForInvalidJSON(t *testing.T) {
	resp := decodeTestConnectionResponse(t, TestModelConnectionInternal(`{invalid`))
	if resp.Success {
		t.Fatalf("expected invalid JSON response to fail: %#v", resp)
	}
	if !strings.Contains(resp.Error, "invalid JSON") {
		t.Fatalf("expected invalid JSON error, got %q", resp.Error)
	}
}

func TestTestModelConnectionInternalReturnsValidationErrorBeforeNetworkCall(t *testing.T) {
	resp := decodeTestConnectionResponse(t, TestModelConnectionInternal(`{"provider":"openai","api_key":"test-key"}`))
	if resp.Success {
		t.Fatalf("expected missing model response to fail: %#v", resp)
	}
	if resp.Error != "OpenAI model name is required" {
		t.Fatalf("unexpected validation error: %q", resp.Error)
	}
}

func TestValidateSecurityModelConfigProviderRequirements(t *testing.T) {
	tests := []struct {
		name      string
		config    repository.SecurityModelConfig
		wantError string
	}{
		{
			name:      "openai requires api key",
			config:    repository.SecurityModelConfig{Provider: "openai", Model: "gpt-test"},
			wantError: "OpenAI API key is required",
		},
		{
			name:      "ollama only requires model",
			config:    repository.SecurityModelConfig{Provider: "ollama", Model: "llama3"},
			wantError: "",
		},
		{
			name:      "compatible requires endpoint",
			config:    repository.SecurityModelConfig{Provider: "openai_compatible", APIKey: "key", Model: "model"},
			wantError: "OpenAI-compatible base URL is required",
		},
		{
			name:      "anthropic compatible requires endpoint",
			config:    repository.SecurityModelConfig{Provider: "anthropic_compatible", APIKey: "key", Model: "model"},
			wantError: "Anthropic-compatible base URL is required",
		},
		{
			name:      "unknown provider falls back to generic api key validation",
			config:    repository.SecurityModelConfig{Provider: "custom", Model: "model"},
			wantError: "custom API key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSecurityModelConfig(&tt.config)
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("expected valid config, got: %v", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantError {
				t.Fatalf("expected error %q, got %v", tt.wantError, err)
			}
		})
	}
}

func TestToJSONStringReturnsMarshalFailureEnvelope(t *testing.T) {
	got := toJSONString(map[string]interface{}{"bad": func() {}})
	if got != `{"success":false,"error":"marshal error"}` {
		t.Fatalf("unexpected marshal failure envelope: %s", got)
	}
}

func decodeTestConnectionResponse(t *testing.T, raw string) TestConnectionResponse {
	t.Helper()

	var resp TestConnectionResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("failed to decode test connection response %q: %v", raw, err)
	}
	return resp
}
