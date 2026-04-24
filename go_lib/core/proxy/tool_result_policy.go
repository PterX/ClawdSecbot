package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	chatmodelrouting "go_lib/chatmodel-routing"
	"go_lib/core/logging"
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

func (shepherdToolResultPolicyHook) Evaluate(ctx context.Context, pp *ProxyProtection, policyCtx toolResultPolicyContext) toolResultPolicyResult {
	if !policyCtx.HasToolResultMessages || pp.shepherdGate == nil {
		return toolResultPolicyResult{}
	}

	pp.ensureRecoveryMutex()
	pp.recoveryMu.Lock()
	armed := pp.pendingRecoveryArmed
	pp.recoveryMu.Unlock()

	if armed {
		pp.clearPendingToolCallRecovery()
		pp.sendTerminalLog("🔄 用户已确认恢复，跳过 ShepherdGate 检测，放行请求")
		pp.sendLog("proxy_tool_result_recovery_allowed", map[string]interface{}{
			"armed": true,
		})
		pp.emitMonitorSecurityDecision("RECOVERY_ALLOWED", "user confirmed recovery", false, "")
		return toolResultPolicyResult{}
	}

	pp.configMu.RLock()
	auditOnlyForShepherd := pp.auditOnly
	pp.configMu.RUnlock()

	if auditOnlyForShepherd {
		logging.Info("[ProxyProtection] Audit-only mode, skipping ShepherdGate analysis")
		pp.sendTerminalLog("📋 仅审计模式，跳过 ShepherdGate 检测，直接放行")
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

		pp.sendTerminalLog(fmt.Sprintf(
			"检测到 ClawdSecbot 沙箱已阻止工具结果，跳过 ShepherdGate 二次确认: tool=%s, tool_call_id=%s",
			tr.FuncName,
			tr.ToolCallID,
		))
		pp.sendLog("proxy_tool_result_sandbox_blocked", map[string]interface{}{
			"tool_id":  tr.ToolCallID,
			"tool":     tr.FuncName,
			"detected": true,
		})

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
	pp.sendTerminalLog(fmt.Sprintf("🔍 ShepherdGate 正在检查 %d 个工具结果: %s", len(toolResultInfos), strings.Join(toolNames, ", ")))

	pp.mu.Lock()
	contextMessages := pp.lastContextMessages
	cachedLastUserMsg := pp.lastUserMessageContent
	pp.mu.Unlock()

	securityModel := pp.shepherdGate.GetModelName()
	logging.Info("[ProxyProtection] ShepherdGate tool result detection triggered: toolCalls=%d, toolResults=%d, securityModel=%s", len(toolCallInfos), len(toolResultInfos), securityModel)

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
		pp.sendTerminalLog(fmt.Sprintf("📊 ShepherdGate Token Usage: %d (Prompt: %d, Completion: %d)",
			decision.Usage.TotalTokens, decision.Usage.PromptTokens, decision.Usage.CompletionTokens))
	}

	if err != nil {
		logging.Error("[ProxyProtection] ShepherdGate tool result check failed: %v, fail-open", err)
		return toolResultPolicyResult{}
	}
	if decision.Status == "ALLOWED" {
		logging.Info("[ProxyProtection] ShepherdGate tool result decision: ALLOWED, tools=%s", strings.Join(toolNames, ", "))
		pp.sendTerminalLog(fmt.Sprintf("✅ ShepherdGate 工具结果检查通过 (ALLOWED): %s", strings.Join(toolNames, ", ")))
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

	logging.Info("[ProxyProtection] ShepherdGate tool result decision: status=%s, reason=%s", decision.Status, decision.Reason)
	pp.sendTerminalLog(fmt.Sprintf("🛡️ ShepherdGate 拦截工具结果: %s - %s", decision.Status, decision.Reason))
	pp.sendLog("proxy_tool_result_decision", map[string]interface{}{
		"status":      decision.Status,
		"reason":      decision.Reason,
		"blocked":     true,
		"skill":       decision.Skill,
		"action_desc": decision.ActionDesc,
		"risk_type":   decision.RiskType,
	})

	pp.storePendingToolCallRecovery(nil, "", decision.Reason, "tool_result")

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
