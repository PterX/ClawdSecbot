package sdk

import (
	"context"
	"testing"

	"go_lib/chatmodel-routing/adapter"

	"github.com/openai/openai-go"
)

type mockProvider struct{}

func (m *mockProvider) ChatCompletion(ctx context.Context, req *openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	return nil, nil
}
func (m *mockProvider) ChatCompletionStream(ctx context.Context, req *openai.ChatCompletionNewParams) (adapter.Stream, error) {
	return nil, nil
}
func (m *mockProvider) ChatCompletionRaw(ctx context.Context, body []byte) (*openai.ChatCompletion, error) {
	return nil, nil
}
func (m *mockProvider) ChatCompletionStreamRaw(ctx context.Context, body []byte) (adapter.Stream, error) {
	return nil, nil
}

func TestRegistry(t *testing.T) {
	Register("mock", func(k string) adapter.Provider { return &mockProvider{} })

	p, err := Get("mock", "key")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if p == nil {
		t.Error("Get returned nil")
	}

	_, err = Get("unknown", "key")
	if err == nil {
		t.Error("Get unknown should fail")
	}
}
