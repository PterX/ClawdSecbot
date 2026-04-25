package proxy

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	chatmodelrouting "go_lib/chatmodel-routing"
	"go_lib/core/shepherd"
)

type toolResultPolicyContext struct {
	RequestID                string
	HasToolResultMessages    bool
	LatestAssistantToolCalls []toolCallRef
	ToolResultsMap           map[string]string
}

type toolResultPolicyResult struct {
	Result  *chatmodelrouting.FilterRequestResult
	Pass    bool
	Handled bool
}

type toolResultPolicyHook interface {
	Name() string
	Evaluate(ctx context.Context, pp *ProxyProtection, policyCtx toolResultPolicyContext) toolResultPolicyResult
}

type shepherdToolResultPolicyHook struct{}

func (shepherdToolResultPolicyHook) Name() string {
	return "shepherd_tool_result"
}

func toolCallIDsFromRefs(refs []toolCallRef) []string {
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		if id := strings.TrimSpace(ref.ID); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func toolCallIDsFromToolResults(results []ToolResultInfo) []string {
	ids := make([]string, 0, len(results))
	for _, result := range results {
		if id := strings.TrimSpace(result.ToolCallID); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func (shepherdToolResultPolicyHook) Evaluate(ctx context.Context, pp *ProxyProtection, policyCtx toolResultPolicyContext) toolResultPolicyResult {
	if !policyCtx.HasToolResultMessages || pp.shepherdGate == nil {
		return toolResultPolicyResult{}
	}

	pp.ensureRecoveryMutex()
	pp.recoveryMu.Lock()
	armed := pp.pendingRecoveryArmed
	pp.recoveryMu.Unlock()

	if armed {
		toolCallIDs := append(pp.pendingRecoveryToolCallIDs(), toolCallIDsFromRefs(policyCtx.LatestAssistantToolCalls)...)
		cleared := pp.clearBlockedToolCallIDs(toolCallIDs)
		pp.clearPendingToolCallRecovery()
		pp.sendSecurityFlowLog(securityFlowStageRecovery, "recovery is armed; skipping tool_result analysis and allowing request cleared_blocked_tool_call_ids=%d", cleared)
		pp.sendLog("proxy_tool_result_recovery_allowed", map[string]interface{}{
			"armed": true,
			"ids":   toolCallIDs,
		})
		pp.emitMonitorSecurityDecision("RECOVERY_ALLOWED", "user confirmed recovery", false, "")
		return toolResultPolicyResult{}
	}

	pp.configMu.RLock()
	auditOnlyForShepherd := pp.auditOnly
	pp.configMu.RUnlock()

	if auditOnlyForShepherd {
		logSecurityFlowInfo(securityFlowStageToolCallResult, "audit_only=true; skipping ShepherdGate analysis")
		pp.sendSecurityFlowLog(securityFlowStageToolCallResult, "audit_only=true; allowing tool results without blocking")
		return toolResultPolicyResult{}
	}

	toolCallInfos := make([]ToolCallInfo, 0, len(policyCtx.LatestAssistantToolCalls))
	for _, tcRef := range policyCtx.LatestAssistantToolCalls {
		info := ToolCallInfo{
			Name:       tcRef.FuncName,
			RawArgs:    tcRef.RawArgs,
			ToolCallID: tcRef.ID,
		}
		if pp.toolValidator != nil {
			info.IsSensitive = pp.toolValidator.IsSensitive(tcRef.FuncName)
		}
		if tcRef.RawArgs != "" {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(tcRef.RawArgs), &args); err == nil {
				info.Arguments = args
			}
		}
		toolCallInfos = append(toolCallInfos, info)
	}

	toolResultInfos := make([]ToolResultInfo, 0, len(policyCtx.LatestAssistantToolCalls))
	for _, tcRef := range policyCtx.LatestAssistantToolCalls {
		tcID := strings.TrimSpace(tcRef.ID)
		if content, ok := policyCtx.ToolResultsMap[tcID]; ok {
			toolResultInfos = append(toolResultInfos, ToolResultInfo{
				ToolCallID: tcRef.ID,
				FuncName:   tcRef.FuncName,
				Content:    content,
			})
		}
	}

	for _, tr := range toolResultInfos {
		if !isClawdSecbotSandboxBlockedToolResult(tr.Content) {
			continue
		}
		if !pp.markSandboxBlockedToolResultIfFirst(tr.ToolCallID) {
			continue
		}

		pp.sendSecurityFlowLog(securityFlowStageToolCallResult,
			"sandbox already blocked tool result; skipping duplicate confirmation: tool=%s tool_call_id=%s",
			tr.FuncName,
			tr.ToolCallID,
		)
		pp.sendLog("proxy_tool_result_sandbox_blocked", map[string]interface{}{
			"tool_id":  tr.ToolCallID,
			"tool":     tr.FuncName,
			"detected": true,
		})
		pp.markBlockedToolCallIDs([]string{tr.ToolCallID})

		sandboxReason := "tool result already blocked by ClawdSecbot sandbox"
		pp.emitMonitorSecurityDecision(
			"SANDBOX_BLOCKED",
			sandboxReason,
			false,
			"",
		)
		pp.emitSecurityEvent(policyCtx.RequestID, "blocked", "Sandbox blocked tool execution", "SANDBOX_BLOCKED", sandboxReason)
		pp.auditLogSafe("set_decision_sandbox_blocked", func(tracker *AuditChainTracker) {
			tracker.SetRequestDecision(
				policyCtx.RequestID,
				"BLOCK",
				"SANDBOX_BLOCKED",
				sandboxReason,
				100,
			)
		})
		return toolResultPolicyResult{Pass: true, Handled: true}
	}

	toolNames := make([]string, 0, len(toolCallInfos))
	for _, tc := range toolCallInfos {
		toolNames = append(toolNames, tc.Name)
	}
	pp.updateTruthRecord(policyCtx.RequestID, func(r *TruthRecord) {
		// Tool names/count will be computed from ToolCalls by frontend getters
	})
	pp.sendSecurityFlowLog(securityFlowStageToolCallResult, "analysis_start: tool_result_count=%d tools=%s", len(toolResultInfos), strings.Join(toolNames, ", "))

	pp.mu.Lock()
	contextMessages := pp.lastContextMessages
	cachedLastUserMsg := pp.lastUserMessageContent
	pp.mu.Unlock()

	securityModel := pp.shepherdGate.GetModelName()
	logSecurityFlowInfo(securityFlowStageToolCallResult, "deep_analysis_triggered: tool_calls=%d tool_results=%d security_model=%s", len(toolCallInfos), len(toolResultInfos), securityModel)

	estimatedSecurityTokens := estimateToolResultSecurityAnalysisTokens(contextMessages, toolCallInfos, toolResultInfos, cachedLastUserMsg)
	budgetDecision := pp.evaluateSecurityBudget(estimatedSecurityTokens)
	if !budgetDecision.Allowed {
		logSecurityFlowWarning(securityFlowStageBudget, "tool_result deep analysis skipped: %s", budgetDecision.Reason)
		pp.sendSecurityFlowLog(securityFlowStageBudget, "tool_result deep analysis skipped: %s", budgetDecision.Reason)
		if fallbackDecision := inspectToolResultBudgetFallback(toolResultInfos, budgetDecision, pp.shepherdGate.EffectiveLanguage()); fallbackDecision != nil {
			toolCallIDs := toolCallIDsFromToolResults(toolResultInfos)
			pp.markBlockedToolCallIDs(toolCallIDs)
			pp.storePendingToolCallRecoveryWithIDs(nil, toolCallIDs, "", fallbackDecision.Reason, "tool_result_budget")
			result, pass := pp.applyRequestSecurityPolicyDecision(ctx, policyCtx.RequestID, *fallbackDecision)
			return toolResultPolicyResult{Result: result, Pass: pass, Handled: true}
		}
		pp.sendLog("proxy_tool_result_security_budget_skipped", map[string]interface{}{
			"request_id":      policyCtx.RequestID,
			"business_tokens": budgetDecision.BusinessTokens,
			"security_tokens": budgetDecision.SecurityTokens,
			"estimated":       budgetDecision.Estimated,
			"limit":           budgetDecision.Limit,
		})
		return toolResultPolicyResult{}
	}

	// Use the proxy lifecycle context instead of the request context so a
	// client-side disconnect does not cancel security analysis mid-flight.
	checkCtx := shepherd.WithBotID(pp.ctx, pp.assetID)
	decision, err := pp.shepherdGate.CheckToolCall(checkCtx, contextMessages, toolCallInfos, toolResultInfos, cachedLastUserMsg, policyCtx.RequestID)

	pp.statsMu.Lock()
	pp.analysisCount++
	pp.statsMu.Unlock()
	pp.sendMetricsToCallback()

	if decision != nil && decision.Usage != nil {
		pp.metricsMu.Lock()
		pp.auditTokens += decision.Usage.TotalTokens
		pp.auditPromptTokens += decision.Usage.PromptTokens
		pp.auditCompletionTokens += decision.Usage.CompletionTokens
		pp.metricsMu.Unlock()
		pp.sendMetricsToCallback()
		pp.sendSecurityFlowLog(securityFlowStageBudget, "analysis_token_usage: total=%d prompt=%d completion=%d",
			decision.Usage.TotalTokens, decision.Usage.PromptTokens, decision.Usage.CompletionTokens)
	}

	if err != nil {
		logSecurityFlowError(securityFlowStageToolCallResult, "analysis_failed: err=%v action=fail_open", err)
		return toolResultPolicyResult{}
	}
	if decision.Status == "ALLOWED" {
		logSecurityFlowInfo(securityFlowStageToolCallResult, "decision: status=ALLOWED tools=%s", strings.Join(toolNames, ", "))
		pp.sendSecurityFlowLog(securityFlowStageToolCallResult, "decision: status=ALLOWED tools=%s", strings.Join(toolNames, ", "))
		pp.sendLog("proxy_tool_result_decision", map[string]interface{}{
			"status":      decision.Status,
			"reason":      decision.Reason,
			"blocked":     false,
			"skill":       decision.Skill,
			"action_desc": decision.ActionDesc,
			"risk_type":   decision.RiskType,
		})
		pp.emitMonitorSecurityDecision(decision.Status, decision.Reason, false, "")
		pp.updateTruthRecord(policyCtx.RequestID, func(r *TruthRecord) {
			r.Decision = &SecurityDecision{
				Action: "ALLOW",
				Reason: decision.Reason,
			}
		})
		pp.auditLogSafe("set_decision_shepherd_allowed", func(tracker *AuditChainTracker) {
			tracker.SetRequestDecision(policyCtx.RequestID, "ALLOW", "", decision.Reason, 0)
		})
		return toolResultPolicyResult{}
	}

	logSecurityFlowInfo(securityFlowStageToolCallResult, "decision: status=%s reason=%s", decision.Status, decision.Reason)
	pp.sendSecurityFlowLog(securityFlowStageToolCallResult, "decision: status=%s reason=%s", decision.Status, decision.Reason)
	pp.sendLog("proxy_tool_result_decision", map[string]interface{}{
		"status":      decision.Status,
		"reason":      decision.Reason,
		"blocked":     true,
		"skill":       decision.Skill,
		"action_desc": decision.ActionDesc,
		"risk_type":   decision.RiskType,
	})

	toolCallIDs := toolCallIDsFromToolResults(toolResultInfos)
	pp.markBlockedToolCallIDs(toolCallIDs)
	pp.storePendingToolCallRecoveryWithIDs(nil, toolCallIDs, "", decision.Reason, "tool_result")

	securityMsg := pp.shepherdGate.FormatSecurityMockReply(decision)
	pp.emitMonitorSecurityDecision(decision.Status, decision.Reason, true, securityMsg)
	recordAction := "BLOCK"
	recordRiskLevel := "BLOCKED"
	if decision.Status == "NEEDS_CONFIRMATION" {
		recordAction = "NEEDS_CONFIRMATION"
		recordRiskLevel = "NEEDS_CONFIRMATION"
	}
	pp.updateTruthRecord(policyCtx.RequestID, func(r *TruthRecord) {
		r.Phase = RecordPhaseStopped
		r.CompletedAt = time.Now().Format(time.RFC3339Nano)
		r.Decision = &SecurityDecision{
			Action:     recordAction,
			RiskLevel:  recordRiskLevel,
			Reason:     decision.Reason,
			Confidence: 100,
		}
		applyRecordPrimaryContent(r, RecordContentSecurity, securityMsg, true)
	})
	pp.statsMu.Lock()
	pp.blockedCount++
	pp.warningCount++
	pp.statsMu.Unlock()
	pp.sendMetricsToCallback()

	shepherdActionDesc := strings.TrimSpace(decision.ActionDesc)
	if shepherdActionDesc == "" {
		shepherdActionDesc = decision.Reason
	}
	shepherdRiskType := strings.TrimSpace(decision.RiskType)
	if shepherdRiskType == "" {
		shepherdRiskType = decision.Status
	}
	shepherdDetail := decision.Reason
	if strings.Contains(decision.Reason, shepherd.PostValidationOverrideTag) {
		shepherdDetail = "post_validation_override | " + decision.Reason
	}
	shepherdEventType := "blocked"
	if decision.Status == "NEEDS_CONFIRMATION" {
		shepherdEventType = "needs_confirmation"
	}
	pp.emitSecurityEvent(policyCtx.RequestID, shepherdEventType, shepherdActionDesc, shepherdRiskType, shepherdDetail)
	pp.emitMonitorResponseReturned(decision.Status, securityMsg, securityMsg)
	pp.auditLogSafe("set_decision_shepherd_blocked", func(tracker *AuditChainTracker) {
		tracker.SetRequestDecision(policyCtx.RequestID, recordAction, recordRiskLevel, decision.Reason, 100)
		tracker.FinalizeRequestOutput(policyCtx.RequestID, securityMsg)
	})
	pp.clearRequestContext(ctx)
	return toolResultPolicyResult{
		Result:  &chatmodelrouting.FilterRequestResult{MockContent: securityMsg},
		Pass:    false,
		Handled: true,
	}
}

func (pp *ProxyProtection) runToolResultPolicyHooks(ctx context.Context, policyCtx toolResultPolicyContext) toolResultPolicyResult {
	hooks := []toolResultPolicyHook{
		shepherdToolResultPolicyHook{},
	}
	for _, hook := range hooks {
		result := hook.Evaluate(ctx, pp, policyCtx)
		if result.Handled {
			return result
		}
	}
	return toolResultPolicyResult{}
}
