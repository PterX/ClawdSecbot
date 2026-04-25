package proxy

import (
	"context"
	"strings"
	"sync"
	"time"

	"go_lib/core/shepherd"

	"github.com/openai/openai-go"
)

type pendingToolCallRecovery struct {
	ToolCalls        []openai.ChatCompletionMessageToolCall
	ToolCallIDs      []string
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

func cloneStringSlice(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = normalizeBlockedToolCallID(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func toolCallIDsFromOpenAIToolCalls(calls []openai.ChatCompletionMessageToolCall) []string {
	ids := make([]string, 0, len(calls))
	for _, call := range calls {
		if id := normalizeBlockedToolCallID(call.ID); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
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
	pp.storePendingToolCallRecoveryWithIDs(toolCalls, toolCallIDsFromOpenAIToolCalls(toolCalls), assistantContent, riskReason, source)
}

func (pp *ProxyProtection) storePendingToolCallRecoveryWithIDs(toolCalls []openai.ChatCompletionMessageToolCall, toolCallIDs []string, assistantContent, riskReason, source string) {
	pp.ensureRecoveryMutex()
	pp.recoveryMu.Lock()
	pp.pendingRecovery = &pendingToolCallRecovery{
		ToolCalls:        cloneToolCalls(toolCalls),
		ToolCallIDs:      cloneStringSlice(toolCallIDs),
		AssistantContent: assistantContent,
		RiskReason:       riskReason,
		Source:           source,
		CreatedAt:        time.Now(),
	}
	pp.pendingRecoveryArmed = false
	pp.recoveryMu.Unlock()
}

func (pp *ProxyProtection) pendingRecoveryToolCallIDs() []string {
	pp.ensureRecoveryMutex()
	pp.recoveryMu.Lock()
	defer pp.recoveryMu.Unlock()
	if pp.pendingRecovery == nil {
		return nil
	}
	return cloneStringSlice(pp.pendingRecovery.ToolCallIDs)
}

func (pp *ProxyProtection) pendingRecoveryRequiresToolResult() bool {
	return len(pp.pendingRecoveryToolCallIDs()) > 0
}

func (pp *ProxyProtection) hasPendingToolCallRecovery() bool {
	pp.ensureRecoveryMutex()
	pp.recoveryMu.Lock()
	defer pp.recoveryMu.Unlock()
	return pp.pendingRecovery != nil
}

func (pp *ProxyProtection) clearPendingToolCallRecovery() {
	pp.ensureRecoveryMutex()
	pp.recoveryMu.Lock()
	pp.pendingRecovery = nil
	pp.pendingRecoveryArmed = false
	pp.recoveryMu.Unlock()
}

func (pp *ProxyProtection) recoverPendingToolCallRecoveryFromHistory(messages []openai.ChatCompletionMessageParamUnion) bool {
	if pp == nil || len(messages) == 0 {
		return false
	}
	if pp.hasPendingToolCallRecovery() {
		return false
	}

	promptIndex, promptContent := latestRecoveryPromptBeforeLatestUser(messages)
	if promptIndex < 0 {
		return false
	}

	toolCallIDs := toolCallIDsImmediatelyBefore(messages, promptIndex)
	reason := recoveryReasonFromPrompt(promptContent)
	if reason == "" {
		reason = "Recovered pending ShepherdGate confirmation from request history."
	}

	pp.storePendingToolCallRecoveryWithIDs(nil, toolCallIDs, promptContent, reason, "request_history")
	if len(toolCallIDs) > 0 {
		pp.markBlockedToolCallIDs(toolCallIDs)
	}
	pp.sendSecurityFlowLog(securityFlowStageRecovery, "recovered pending confirmation from request history: tool_call_ids=%v", toolCallIDs)
	return true
}

func latestRecoveryPromptBeforeLatestUser(messages []openai.ChatCompletionMessageParamUnion) (int, string) {
	latestUserIndex := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.EqualFold(getMessageRole(messages[i]), "user") {
			latestUserIndex = i
			break
		}
	}
	if latestUserIndex <= 0 {
		return -1, ""
	}

	for i := latestUserIndex - 1; i >= 0; i-- {
		if !strings.EqualFold(getMessageRole(messages[i]), "assistant") {
			continue
		}
		content := extractMessageContent(messages[i])
		if isShepherdGateRecoveryPromptContent(content) {
			return i, content
		}
		return -1, ""
	}
	return -1, ""
}

func isShepherdGateRecoveryPromptContent(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" || !strings.Contains(content, "[ShepherdGate]") {
		return false
	}
	upper := strings.ToUpper(content)
	lower := strings.ToLower(content)
	return strings.Contains(upper, "NEEDS_CONFIRMATION") ||
		strings.Contains(content, "需要确认") ||
		strings.Contains(content, "继续可回复") ||
		strings.Contains(lower, "continue replies:")
}

func toolCallIDsImmediatelyBefore(messages []openai.ChatCompletionMessageParamUnion, promptIndex int) []string {
	if promptIndex <= 0 || promptIndex > len(messages) {
		return nil
	}
	ids := make([]string, 0)
	seen := make(map[string]struct{})
	for i := promptIndex - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.OfTool == nil {
			break
		}
		toolCallID := normalizeBlockedToolCallID(msg.OfTool.ToolCallID)
		if toolCallID == "" {
			continue
		}
		if _, ok := seen[toolCallID]; ok {
			continue
		}
		seen[toolCallID] = struct{}{}
		ids = append(ids, toolCallID)
	}
	return ids
}

func recoveryReasonFromPrompt(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if idx := strings.LastIndex(line, "原因:"); idx >= 0 {
			return strings.TrimSpace(line[idx+len("原因:"):])
		}
		if idx := strings.LastIndex(line, "Reason:"); idx >= 0 {
			return strings.TrimSpace(line[idx+len("Reason:"):])
		}
	}
	return ""
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
		logSecurityFlowInfo(securityFlowStageRecovery, "pending recovery already armed; waiting for injection")
		return true
	}
	snapshot := *pp.pendingRecovery
	snapshot.ToolCalls = cloneToolCalls(pp.pendingRecovery.ToolCalls)
	snapshot.ToolCallIDs = cloneStringSlice(pp.pendingRecovery.ToolCallIDs)
	pp.recoveryMu.Unlock()

	if pp.shepherdGate == nil {
		logSecurityFlowWarning(securityFlowStageRecovery, "shepherdGate is nil; skipping recovery intent analysis")
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
		logSecurityFlowWarning(securityFlowStageRecovery, "intent analysis failed: %v", err)
		return false
	}
	if intentDecision == nil {
		logSecurityFlowWarning(securityFlowStageRecovery, "intent analysis returned nil decision")
		return false
	}

	intent := shepherd.NormalizeRecoveryIntent(intentDecision.Intent)
	logSecurityFlowInfo(securityFlowStageRecovery, "intent_decision: intent=%s reason=%s", intent, intentDecision.Reason)

	pp.recoveryMu.Lock()
	defer pp.recoveryMu.Unlock()
	if pp.pendingRecovery == nil || !pp.pendingRecovery.CreatedAt.Equal(snapshot.CreatedAt) {
		logSecurityFlowInfo(securityFlowStageRecovery, "pending recovery changed before applying intent decision; skipping")
		return false
	}

	switch intent {
	case "CONFIRM":
		pp.pendingRecoveryArmed = true
		logSecurityFlowInfo(securityFlowStageRecovery, "user confirmation recognized; recovery armed")
		return true
	case "REJECT":
		pp.pendingRecovery = nil
		pp.pendingRecoveryArmed = false
		logSecurityFlowInfo(securityFlowStageRecovery, "user rejection recognized; pending recovery cleared")
		return false
	default:
		logSecurityFlowInfo(securityFlowStageRecovery, "no clear confirmation or rejection detected; keeping pending recovery")
		return false
	}
}
