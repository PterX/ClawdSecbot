package shepherd

import "testing"

func TestMergeUsage(t *testing.T) {
	left := &Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}
	right := &Usage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5}

	got := mergeUsage(left, right)
	if got == nil {
		t.Fatalf("expected merged usage")
	}
	if got.PromptTokens != 13 || got.CompletionTokens != 7 || got.TotalTokens != 20 {
		t.Fatalf("unexpected merged usage: %+v", got)
	}
}

func TestMergeUsageNilCases(t *testing.T) {
	if got := mergeUsage(nil, nil); got != nil {
		t.Fatalf("expected nil when both nil")
	}

	one := &Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2}
	got := mergeUsage(one, nil)
	if got == nil || got.TotalTokens != 2 {
		t.Fatalf("unexpected merge left-only: %+v", got)
	}
}
