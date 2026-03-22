package proxy

import (
	"context"
	"sync"
	"time"

	"go_lib/core/logging"
	"go_lib/core/shepherd"

	"github.com/openai/openai-go"
)

type pendingToolCallRecovery struct {
	ToolCalls        []openai.ChatCompletionMessageToolCall
	AssistantContent string
	RiskReason       string
	Source           string
	CreatedAt        time.Time
}

func cloneToolCalls(calls []openai.ChatCompletionMessageToolCall) []openai.ChatCompletionMessageToolCall {
	out := make([]openai.ChatCompletionMessageToolCall, len(calls))
	copy(out, calls)
	return out
}

func extractRecentConversationMessagesFromParams(messages []openai.ChatCompletionMessageParamUnion, limit int) []ConversationMessage {
	if limit <= 0 {
		limit = 50
	}
	start := 0
	if len(messages) > limit {
		start = len(messages) - limit
	}
	out := make([]ConversationMessage, 0, len(messages)-start)
	for _, msg := range messages[start:] {
		out = append(out, extractConversationMessage(msg))
	}
	return out
}

func (pp *ProxyProtection) ensureRecoveryMutex() {
	if pp.recoveryMu == nil {
		pp.recoveryMu = &sync.Mutex{}
	}
}

func (pp *ProxyProtection) storePendingToolCallRecovery(toolCalls []openai.ChatCompletionMessageToolCall, assistantContent, riskReason, source string) {
	pp.ensureRecoveryMutex()
	pp.recoveryMu.Lock()
	pp.pendingRecovery = &pendingToolCallRecovery{
		ToolCalls:        cloneToolCalls(toolCalls),
		AssistantContent: assistantContent,
		RiskReason:       riskReason,
		Source:           source,
		CreatedAt:        time.Now(),
	}
	pp.pendingRecoveryArmed = false
	pp.recoveryMu.Unlock()
}

func (pp *ProxyProtection) clearPendingToolCallRecovery() {
	pp.ensureRecoveryMutex()
	pp.recoveryMu.Lock()
	pp.pendingRecovery = nil
	pp.pendingRecoveryArmed = false
	pp.recoveryMu.Unlock()
}

func (pp *ProxyProtection) armPendingRecoveryFromRequest(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion) bool {
	contextMessages := extractRecentConversationMessagesFromParams(messages, 80)
	if len(contextMessages) == 0 {
		return false
	}
	return pp.armPendingRecoveryFromContext(ctx, contextMessages)
}

func (pp *ProxyProtection) armPendingRecoveryFromContext(ctx context.Context, contextMessages []ConversationMessage) bool {
	pp.ensureRecoveryMutex()
	pp.recoveryMu.Lock()
	if pp.pendingRecovery == nil {
		pp.recoveryMu.Unlock()
		return false
	}
	if pp.pendingRecoveryArmed {
		pp.recoveryMu.Unlock()
		logging.Info("[ShepherdGate][Recovery] Pending recovery already armed, waiting for injection.")
		return true
	}
	snapshot := *pp.pendingRecovery
	snapshot.ToolCalls = cloneToolCalls(pp.pendingRecovery.ToolCalls)
	pp.recoveryMu.Unlock()

	if pp.shepherdGate == nil {
		logging.Warning("[ShepherdGate][Recovery] shepherdGate is nil, skip recovery intent analysis.")
		return false
	}

	toolCalls := extractToolCalls(snapshot.ToolCalls)
	intentDecision, err := pp.shepherdGate.EvaluateRecoveryIntent(ctx, contextMessages, toolCalls, snapshot.RiskReason)
	if intentDecision != nil && intentDecision.Usage != nil {
		pp.metricsMu.Lock()
		pp.auditTokens += intentDecision.Usage.TotalTokens
		pp.auditPromptTokens += intentDecision.Usage.PromptTokens
		pp.auditCompletionTokens += intentDecision.Usage.CompletionTokens
		pp.metricsMu.Unlock()
		pp.sendMetricsToCallback()
	}
	if err != nil {
		logging.Warning("[ShepherdGate][Recovery] Recovery intent analysis failed: %v", err)
		return false
	}
	if intentDecision == nil {
		logging.Warning("[ShepherdGate][Recovery] Recovery intent analysis returned nil decision.")
		return false
	}

	intent := shepherd.NormalizeRecoveryIntent(intentDecision.Intent)
	logging.Info("[ShepherdGate][Recovery] Recovery intent decision: intent=%s, reason=%s", intent, intentDecision.Reason)

	pp.recoveryMu.Lock()
	defer pp.recoveryMu.Unlock()
	if pp.pendingRecovery == nil || !pp.pendingRecovery.CreatedAt.Equal(snapshot.CreatedAt) {
		logging.Info("[ShepherdGate][Recovery] Pending recovery changed before applying intent decision, skip.")
		return false
	}

	switch intent {
	case "CONFIRM":
		pp.pendingRecoveryArmed = true
		logging.Info("[ShepherdGate][Recovery] User confirmation recognized by security agent, recovery armed.")
		return true
	case "REJECT":
		pp.pendingRecovery = nil
		pp.pendingRecoveryArmed = false
		logging.Info("[ShepherdGate][Recovery] User rejection recognized by security agent, pending recovery cleared.")
		return false
	default:
		logging.Info("[ShepherdGate][Recovery] No clear confirmation/rejection detected, keep pending recovery.")
		return false
	}
}
