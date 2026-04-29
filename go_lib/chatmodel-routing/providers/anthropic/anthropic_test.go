package anthropic

import (
	"encoding/json"
	"testing"

	"github.com/tidwall/gjson"
)

func TestConvertRequestRawMovesSystemMessageAndSetsDefaults(t *testing.T) {
	provider := New("test-key")
	body := []byte(`{
		"model":"claude-test",
		"messages":[
			{"role":"system","content":"follow policy"},
			{"role":"user","content":"hello"}
		]
	}`)

	converted, err := provider.convertRequestRaw(body, false)
	if err != nil {
		t.Fatalf("convertRequestRaw failed: %v", err)
	}

	if got := gjson.GetBytes(converted, "system").String(); got != "follow policy" {
		t.Fatalf("Expected system prompt to be moved, got %q in %s", got, converted)
	}
	if got := gjson.GetBytes(converted, "messages.0.role").String(); got != "user" {
		t.Fatalf("Expected first Anthropic message to be user, got %q in %s", got, converted)
	}
	if got := gjson.GetBytes(converted, "max_tokens").Int(); got == 0 {
		t.Fatalf("Expected max_tokens default, got %d in %s", got, converted)
	}
}

func TestConvertRequestRawEnablesStream(t *testing.T) {
	provider := New("test-key")

	converted, err := provider.convertRequestRaw([]byte(`{"model":"claude-test","messages":[]}`), true)
	if err != nil {
		t.Fatalf("convertRequestRaw failed: %v", err)
	}
	if got := gjson.GetBytes(converted, "stream").Bool(); !got {
		t.Fatalf("Expected stream=true in %s", converted)
	}
}

func TestProviderBaseURLAccessors(t *testing.T) {
	provider := New("test-key")
	if provider.DefaultBaseURL() != defaultBaseURL {
		t.Fatalf("Expected default base URL %q, got %q", defaultBaseURL, provider.DefaultBaseURL())
	}
	if provider.GetBaseURL() != defaultBaseURL {
		t.Fatalf("Expected initial base URL %q, got %q", defaultBaseURL, provider.GetBaseURL())
	}

	customURL := "https://example.test/v1/messages"
	provider.SetBaseURL(customURL)
	if provider.GetBaseURL() != customURL {
		t.Fatalf("Expected custom base URL %q, got %q", customURL, provider.GetBaseURL())
	}
}

func TestConvertResponseMapsAnthropicMessage(t *testing.T) {
	provider := New("test-key")
	resp, err := provider.convertResponse([]byte(`{
		"id":"msg_test",
		"type":"message",
		"model":"claude-test",
		"role":"assistant",
		"content":[{"type":"text","text":"hello"}],
		"stop_reason":"end_turn",
		"usage":{"input_tokens":3,"output_tokens":4}
	}`))
	if err != nil {
		t.Fatalf("convertResponse failed: %v", err)
	}
	if resp.ID != "msg_test" {
		t.Fatalf("Expected response ID msg_test, got %q", resp.ID)
	}
	if len(resp.Choices) != 1 || resp.Choices[0].Message.Content != "hello" {
		body, _ := json.Marshal(resp)
		t.Fatalf("Expected assistant content hello, got %s", body)
	}
}
