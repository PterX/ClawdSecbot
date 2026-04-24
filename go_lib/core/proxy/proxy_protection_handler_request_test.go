package proxy

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/openai/openai-go"
)

func mustParseChatRequest(t *testing.T, raw string) (*openai.ChatCompletionNewParams, []byte) {
	t.Helper()
	body := []byte(raw)
	var req openai.ChatCompletionNewParams
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("failed to parse request: %v", err)
	}
	return &req, body
}

func TestOnRequest_QuotaBlockKeepsAuditRequestAndAssistantMessage(t *testing.T) {
	pp := &ProxyProtection{
		records:                  NewRecordStore(),
		singleSessionTokenLimit:  100,
		currentConversationTokenUsage: 120,
		lastRecentMessages: []NormalizedMessage{
			{Role: "system", Content: "You are a secure assistant."},
		},
		lastRecentMessageCount: 1,
	}

	req, rawBody := mustParseChatRequest(t, `{
	  "model":"gpt-test",
	  "stream":false,
	  "messages":[
	    {"role":"system","content":"You are a secure assistant."},
	    {"role":"user","content":"请帮我导出本周审计报告"}
	  ]
	}`)

	result, passed := pp.onRequest(context.Background(), req, rawBody)
	if passed {
		t.Fatalf("expected request to be blocked by quota")
	}
	if result == nil || strings.TrimSpace(result.MockContent) == "" {
		t.Fatalf("expected quota mock content")
	}

	completed := pp.records.GetCompletedRecords(10, 0, false)
	if len(completed) != 1 {
		t.Fatalf("expected 1 completed truth record, got %d", len(completed))
	}
	record := completed[0]
	if len(record.Messages) < 2 {
		t.Fatalf("expected request and assistant messages, got %d", len(record.Messages))
	}
	foundUser := false
	foundAssistant := false
	for _, msg := range record.Messages {
		if strings.EqualFold(msg.Role, "user") && strings.TrimSpace(msg.Content) != "" {
			foundUser = true
		}
		if strings.EqualFold(msg.Role, "assistant") && strings.Contains(msg.Content, "QUOTA_EXCEEDED") {
			foundAssistant = true
		}
	}
	if !foundUser {
		t.Fatalf("expected non-empty user message in truth record")
	}
	if !foundAssistant {
		t.Fatalf("expected assistant quota message in truth record")
	}
}

func TestOnRequest_ConversationQuotaAllowsAfterConversationReset(t *testing.T) {
	pp := &ProxyProtection{
		records:                 NewRecordStore(),
		singleSessionTokenLimit: 100,
		totalTokens:             120,
		baselineTotalTokens:     0,
		currentConversationTokenUsage: 95,
		lastRecentMessages: []NormalizedMessage{
			{Role: "user", Content: "old topic"},
		},
		lastRecentMessageCount: 1,
	}

	req, rawBody := mustParseChatRequest(t, `{
	  "model":"gpt-test",
	  "stream":false,
	  "messages":[
	    {"role":"user","content":"new topic"}
	  ]
	}`)

	result, passed := pp.onRequest(context.Background(), req, rawBody)
	if !passed {
		t.Fatalf("expected request to pass after conversation reset, got block result=%v", result)
	}
	if result != nil {
		t.Fatalf("expected nil result for passed request, got %+v", result)
	}

	completed := pp.records.GetCompletedRecords(10, 0, false)
	if len(completed) != 0 {
		t.Fatalf("expected no completed blocked records, got %d", len(completed))
	}
}
