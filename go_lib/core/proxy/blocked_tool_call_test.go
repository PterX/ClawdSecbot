package proxy

import (
	"context"
	"sync"
	"testing"
	"time"

	"go_lib/core/shepherd"
)

func TestBlockedToolCallIDsExpire(t *testing.T) {
	pp := &ProxyProtection{}
	now := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	pp.markBlockedToolCallIDsAt([]string{" call_1 "}, now, time.Minute)

	if !pp.isBlockedToolCallIDAt("call_1", now.Add(30*time.Second)) {
		t.Fatalf("expected call_1 to be blocked before expiry")
	}
	if pp.isBlockedToolCallIDAt("call_1", now.Add(time.Minute)) {
		t.Fatalf("expected call_1 to expire at ttl boundary")
	}
}

func TestClearBlockedToolCallIDs(t *testing.T) {
	pp := &ProxyProtection{}
	now := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	pp.markBlockedToolCallIDsAt([]string{"call_1", "call_2"}, now, time.Hour)

	if cleared := pp.clearBlockedToolCallIDs([]string{"call_1"}); cleared != 1 {
		t.Fatalf("expected 1 cleared id, got %d", cleared)
	}
	if pp.isBlockedToolCallIDAt("call_1", now) {
		t.Fatalf("expected call_1 to be cleared")
	}
	if !pp.isBlockedToolCallIDAt("call_2", now) {
		t.Fatalf("expected call_2 to remain blocked")
	}
}

func TestToolResultPolicyConfirmedRecoveryClearsBlockedToolCallIDs(t *testing.T) {
	pp := &ProxyProtection{
		recoveryMu:           &sync.Mutex{},
		pendingRecoveryArmed: true,
		pendingRecovery:      &pendingToolCallRecovery{CreatedAt: time.Now()},
		shepherdGate:         &shepherd.ShepherdGate{},
		blockedToolCallIDs:   make(map[string]time.Time),
	}
	pp.markBlockedToolCallIDsAt([]string{"call_confirmed"}, time.Now(), time.Hour)

	result := pp.runToolResultPolicyHooks(context.Background(), toolResultPolicyContext{
		RequestID:             "req_confirmed",
		HasToolResultMessages: true,
		LatestAssistantToolCalls: []toolCallRef{
			{ID: "call_confirmed", FuncName: "exec"},
		},
		ToolResultsMap: map[string]string{"call_confirmed": "ok"},
	})
	if result.Handled {
		t.Fatalf("expected confirmed recovery to continue normal flow")
	}
	if pp.isBlockedToolCallID("call_confirmed") {
		t.Fatalf("expected confirmed recovery to clear blocked tool_call_id")
	}
}
