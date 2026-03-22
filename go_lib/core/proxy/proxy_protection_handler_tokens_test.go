package proxy

import (
	"context"
	"testing"

	"github.com/openai/openai-go"
)

func TestOnResponse_EstimatesUsageWhenMissing(t *testing.T) {
	pp := &ProxyProtection{
		streamBuffer: &StreamBuffer{},
	}
	pp.streamBuffer.requestMessages = []ConversationMessage{
		{
			Role:    "user",
			Content: "hello, please summarize this text",
		},
	}

	resp := &openai.ChatCompletion{
		Model: "gpt-test",
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Content: "summary output",
				},
			},
		},
		// Usage intentionally omitted (all zero).
	}

	if !pp.onResponse(context.Background(), resp) {
		t.Fatalf("expected onResponse to pass")
	}

	pp.metricsMu.Lock()
	defer pp.metricsMu.Unlock()

	if pp.totalPromptTokens <= 0 {
		t.Fatalf("expected prompt tokens to be estimated, got %d", pp.totalPromptTokens)
	}
	if pp.totalCompletionTokens <= 0 {
		t.Fatalf("expected completion tokens to be estimated, got %d", pp.totalCompletionTokens)
	}
	if pp.totalTokens != pp.totalPromptTokens+pp.totalCompletionTokens {
		t.Fatalf("expected total=%d+%d, got %d", pp.totalPromptTokens, pp.totalCompletionTokens, pp.totalTokens)
	}
}

func TestOnStreamChunk_UsageAccumulatesByDeltaForCumulativeReports(t *testing.T) {
	pp := &ProxyProtection{
		streamBuffer: &StreamBuffer{},
	}

	chunk1 := &openai.ChatCompletionChunk{
		Usage: openai.CompletionUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}
	chunk2 := &openai.ChatCompletionChunk{
		Usage: openai.CompletionUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}
	chunk3 := &openai.ChatCompletionChunk{
		Usage: openai.CompletionUsage{
			PromptTokens:     12,
			CompletionTokens: 7,
			TotalTokens:      19,
		},
	}

	_ = pp.onStreamChunk(context.Background(), chunk1)
	_ = pp.onStreamChunk(context.Background(), chunk2)
	_ = pp.onStreamChunk(context.Background(), chunk3)

	pp.metricsMu.Lock()
	defer pp.metricsMu.Unlock()

	if pp.totalPromptTokens != 12 {
		t.Fatalf("expected prompt tokens=12, got %d", pp.totalPromptTokens)
	}
	if pp.totalCompletionTokens != 7 {
		t.Fatalf("expected completion tokens=7, got %d", pp.totalCompletionTokens)
	}
	if pp.totalTokens != 19 {
		t.Fatalf("expected total tokens=19, got %d", pp.totalTokens)
	}
}
