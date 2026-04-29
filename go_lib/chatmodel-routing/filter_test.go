package chatmodelrouting

import (
	"context"
	"errors"
	"testing"

	"github.com/openai/openai-go"
)

func TestCallbackFilterDefaultAllowsWithoutCallbacks(t *testing.T) {
	filter := NewCallbackFilter(nil, nil, nil)

	if result, pass := filter.FilterRequest(context.Background(), []byte(`{"model":"gpt-test","messages":[]}`)); !pass || result != nil {
		t.Fatalf("Expected request to pass with nil result, got pass=%v result=%v", pass, result)
	}
	if pass := filter.FilterResponse(context.Background(), &openai.ChatCompletion{}); !pass {
		t.Fatal("Expected response to pass without callback")
	}
	if pass := filter.FilterStreamChunk(context.Background(), &openai.ChatCompletionChunk{}); !pass {
		t.Fatal("Expected stream chunk to pass without callback")
	}
}

func TestCallbackFilterInvokesRequestCallbackWithParsedBody(t *testing.T) {
	expectedRaw := []byte(`{"model":"gpt-test","messages":[{"role":"user","content":"hello"}]}`)
	expectedContextValue := errors.New("ctx marker")
	ctx := context.WithValue(context.Background(), expectedContextValue, "seen")
	called := false

	filter := NewCallbackFilter(func(ctx context.Context, req *openai.ChatCompletionNewParams, rawBody []byte) (*FilterRequestResult, bool) {
		called = true
		if req == nil {
			t.Fatal("Expected parsed request")
		}
		if string(req.Model) != "gpt-test" {
			t.Fatalf("Expected parsed model gpt-test, got %q", req.Model)
		}
		if len(req.Messages) != 1 {
			t.Fatalf("Expected one parsed message, got %#v", req.Messages)
		}
		if string(rawBody) != string(expectedRaw) {
			t.Fatalf("Expected raw body %s, got %s", expectedRaw, rawBody)
		}
		if ctx.Value(expectedContextValue) != "seen" {
			t.Fatal("Expected original context to be passed through")
		}
		return &FilterRequestResult{MockContent: "blocked"}, false
	}, nil, nil)

	result, pass := filter.FilterRequest(ctx, expectedRaw)
	if pass {
		t.Fatal("Expected callback to block the request")
	}
	if !called {
		t.Fatal("Expected request callback to be called")
	}
	if result == nil || result.MockContent != "blocked" {
		t.Fatalf("Expected mock block content, got %#v", result)
	}
}

func TestCallbackFilterAllowsInvalidRequestBody(t *testing.T) {
	called := false
	filter := NewCallbackFilter(func(context.Context, *openai.ChatCompletionNewParams, []byte) (*FilterRequestResult, bool) {
		called = true
		return nil, false
	}, nil, nil)

	result, pass := filter.FilterRequest(context.Background(), []byte(`{invalid`))
	if !pass || result != nil {
		t.Fatalf("Expected invalid OpenAI body to pass, got pass=%v result=%v", pass, result)
	}
	if called {
		t.Fatal("Expected request callback not to run for invalid JSON")
	}
}

func TestCallbackFilterResponseCallbacks(t *testing.T) {
	respCalled := false
	chunkCalled := false
	resp := &openai.ChatCompletion{}
	chunk := &openai.ChatCompletionChunk{}

	filter := NewCallbackFilter(nil,
		func(_ context.Context, got *openai.ChatCompletion) bool {
			respCalled = true
			return got == resp
		},
		func(_ context.Context, got *openai.ChatCompletionChunk) bool {
			chunkCalled = true
			return got == chunk
		},
	)

	if !filter.FilterResponse(context.Background(), resp) {
		t.Fatal("Expected response callback to allow original response")
	}
	if !filter.FilterStreamChunk(context.Background(), chunk) {
		t.Fatal("Expected stream callback to allow original chunk")
	}
	if !respCalled || !chunkCalled {
		t.Fatalf("Expected both callbacks to run, resp=%v chunk=%v", respCalled, chunkCalled)
	}
}
