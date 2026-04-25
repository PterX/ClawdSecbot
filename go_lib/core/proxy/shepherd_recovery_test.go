package proxy

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go_lib/core/repository"
	"go_lib/core/shepherd"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/openai/openai-go"
)

type stubChatModelForProxy struct {
	generateResp *schema.Message
	generateErr  error
	called       bool
}

func (m *stubChatModelForProxy) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	m.called = true
	if m.generateErr != nil {
		return nil, m.generateErr
	}
	if m.generateResp != nil {
		return m.generateResp, nil
	}
	return &schema.Message{}, nil
}

func (m *stubChatModelForProxy) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("not implemented in tests")
}

func (m *stubChatModelForProxy) BindTools(_ []*schema.ToolInfo) error {
	return nil
}

func TestArmPendingRecoveryFromContext_Confirm(t *testing.T) {
	pp := &ProxyProtection{
		recoveryMu: &sync.Mutex{},
		shepherdGate: shepherd.NewShepherdGateForTesting(
			&stubChatModelForProxy{
				generateResp: &schema.Message{
					Content: `{"intent":"confirm","reason":"用户已明确确认继续执行。","usage":{"prompt_tokens":9,"completion_tokens":5,"total_tokens":14}}`,
				},
			},
			"zh",
			&repository.SecurityModelConfig{Model: "MiniMax-M2.5"},
		),
		pendingRecovery: &pendingToolCallRecovery{
			ToolCalls: []openai.ChatCompletionMessageToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: openai.ChatCompletionMessageToolCallFunction{
						Name:      "delete_email",
						Arguments: `{"email_id":"m1"}`,
					},
				},
			},
			RiskReason: "delete action requires confirmation",
			CreatedAt:  time.Now(),
		},
	}

	ok := pp.armPendingRecoveryFromContext(context.Background(), []ConversationMessage{
		{Role: "assistant", Content: "[ShepherdGate] 状态: NEEDS_CONFIRMATION"},
		{Role: "user", Content: "确定，继续执行"},
	})
	if !ok {
		t.Fatalf("expected recovery to be armed by security agent confirmation")
	}
	if !pp.pendingRecoveryArmed {
		t.Fatalf("expected pendingRecoveryArmed=true")
	}
}

func TestArmPendingRecoveryFromContext_Reject(t *testing.T) {
	pp := &ProxyProtection{
		recoveryMu: &sync.Mutex{},
		shepherdGate: shepherd.NewShepherdGateForTesting(
			&stubChatModelForProxy{
				generateResp: &schema.Message{
					Content: `{"intent":"REJECT","reason":"用户明确取消执行。","usage":{"prompt_tokens":10,"completion_tokens":6,"total_tokens":16}}`,
				},
			},
			"zh",
			&repository.SecurityModelConfig{Model: "MiniMax-M2.5"},
		),
		pendingRecovery: &pendingToolCallRecovery{
			ToolCalls: []openai.ChatCompletionMessageToolCall{
				{ID: "call_1"},
			},
			CreatedAt: time.Now(),
		},
	}

	ok := pp.armPendingRecoveryFromContext(context.Background(), []ConversationMessage{
		{Role: "assistant", Content: "[ShepherdGate] 状态: NEEDS_CONFIRMATION"},
		{Role: "user", Content: "取消，不要执行"},
	})
	if ok {
		t.Fatalf("expected reject to prevent arming")
	}
	if pp.pendingRecovery != nil {
		t.Fatalf("expected pending recovery cleared on reject")
	}
}

func TestPendingToolRecoveryArming(t *testing.T) {
	pp := &ProxyProtection{}

	toolCalls := []openai.ChatCompletionMessageToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: openai.ChatCompletionMessageToolCallFunction{
				Name:      "delete_email",
				Arguments: `{"email_id":"a1"}`,
			},
		},
	}
	pp.storePendingToolCallRecovery(toolCalls, "assistant tool call", "risk reason", "non_stream")

	// Verify recovery is stored but not armed
	pp.ensureRecoveryMutex()
	pp.recoveryMu.Lock()
	if pp.pendingRecovery == nil {
		t.Fatalf("expected pending recovery to be stored")
	}
	if pp.pendingRecoveryArmed {
		t.Fatalf("expected pending recovery NOT to be armed yet")
	}
	pp.recoveryMu.Unlock()

	// Simulate arming (user confirmation would trigger this)
	pp.recoveryMu.Lock()
	pp.pendingRecoveryArmed = true
	pp.recoveryMu.Unlock()

	// Verify armed state
	pp.recoveryMu.Lock()
	armed := pp.pendingRecoveryArmed
	pp.recoveryMu.Unlock()
	if !armed {
		t.Fatalf("expected pending recovery to be armed")
	}

	// Clear recovery (as onRequest would do when armed)
	pp.clearPendingToolCallRecovery()

	pp.recoveryMu.Lock()
	if pp.pendingRecovery != nil {
		t.Fatalf("expected pending recovery to be cleared")
	}
	if pp.pendingRecoveryArmed {
		t.Fatalf("expected armed flag to be cleared")
	}
	pp.recoveryMu.Unlock()
}
