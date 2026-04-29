package xai

import "testing"

func TestNewConfiguresXAIEndpoint(t *testing.T) {
	provider := New("test-key")
	want := "https://api.x.ai/v1/chat/completions"
	if provider.GetBaseURL() != want {
		t.Fatalf("Expected xAI endpoint %q, got %q", want, provider.GetBaseURL())
	}
}
