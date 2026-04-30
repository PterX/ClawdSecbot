package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatCompletionRawSendsHeadersAndParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Expected authorization header, got %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Expected content type application/json, got %q", got)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		if body["model"] != "gpt-test" {
			t.Fatalf("Expected request model gpt-test, got %#v", body["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_test","object":"chat.completion","created":1,"model":"gpt-test","choices":[]}`))
	}))
	defer server.Close()

	provider := New("test-key")
	provider.SetBaseURL(server.URL)

	resp, err := provider.ChatCompletionRaw(context.Background(), []byte(`{"model":"gpt-test","messages":[]}`))
	if err != nil {
		t.Fatalf("ChatCompletionRaw failed: %v", err)
	}
	if resp.ID != "chatcmpl_test" {
		t.Fatalf("Expected response ID chatcmpl_test, got %q", resp.ID)
	}
	if resp.Model != "gpt-test" {
		t.Fatalf("Expected response model gpt-test, got %q", resp.Model)
	}
}

func TestChatCompletionRawReturnsAPIErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	provider := New("test-key")
	provider.SetBaseURL(server.URL)

	if _, err := provider.ChatCompletionRaw(context.Background(), []byte(`{}`)); err == nil {
		t.Fatal("Expected API error")
	}
}
