package deepseek

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatCompletionRawPreservesReasoningContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Expected authorization header, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl_deepseek",
			"object":"chat.completion",
			"created":1,
			"model":"deepseek-reasoner",
			"choices":[
				{"index":0,"message":{"role":"assistant","content":"answer","reasoning_content":"thinking"},"finish_reason":"stop"}
			]
		}`))
	}))
	defer server.Close()

	provider := New("test-key")
	provider.SetBaseURL(server.URL)

	resp, err := provider.ChatCompletionRaw(context.Background(), []byte(`{"model":"deepseek-reasoner","messages":[]}`))
	if err != nil {
		t.Fatalf("ChatCompletionRaw failed: %v", err)
	}

	field, ok := resp.Choices[0].Message.JSON.ExtraFields["reasoning_content"]
	if !ok {
		t.Fatalf("Expected reasoning_content extra field, got %#v", resp.Choices[0].Message.JSON.ExtraFields)
	}
	if got := field.Raw(); got != `"thinking"` {
		t.Fatalf("Expected reasoning_content to be preserved, got %q", got)
	}
}

func TestChatCompletionStreamRawAddsStreamOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		if body["stream"] != true {
			t.Fatalf("Expected stream=true, got %#v", body["stream"])
		}
		streamOptions, ok := body["stream_options"].(map[string]interface{})
		if !ok || streamOptions["include_usage"] != true {
			t.Fatalf("Expected stream_options.include_usage=true, got %#v", body["stream_options"])
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	provider := New("test-key")
	provider.SetBaseURL(server.URL)

	stream, err := provider.ChatCompletionStreamRaw(context.Background(), []byte(`{"model":"deepseek-chat","messages":[]}`))
	if err != nil {
		t.Fatalf("ChatCompletionStreamRaw failed: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("Failed to close stream: %v", err)
	}
}
