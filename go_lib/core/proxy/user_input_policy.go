package proxy

import (
	"context"
	"fmt"
	"strings"

	chatmodelrouting "go_lib/chatmodel-routing"
	"go_lib/core/shepherd"
	"go_lib/core/skillscan"

	"github.com/openai/openai-go"
)

type userInputPolicyContext struct {
	RequestID string
	Messages  []openai.ChatCompletionMessageParamUnion
}

type userInputPolicyResult struct {
	Result  *chatmodelrouting.FilterRequestResult
	Pass    bool
	Handled bool
}

type userInputPolicyHook interface {
	Name() string
	Evaluate(ctx context.Context, pp *ProxyProtection, policyCtx userInputPolicyContext) userInputPolicyResult
}

type shepherdUserInputPolicyHook struct{}

func (shepherdUserInputPolicyHook) Name() string {
	return "shepherd_user_input"
}

func (shepherdUserInputPolicyHook) Evaluate(ctx context.Context, pp *ProxyProtection, policyCtx userInputPolicyContext) userInputPolicyResult {
	if len(policyCtx.Messages) == 0 || getMessageRole(policyCtx.Messages[len(policyCtx.Messages)-1]) != "user" {
		return userInputPolicyResult{}
	}

	userText := collectUserInputText(policyCtx.Messages)
	if strings.TrimSpace(userText) == "" {
		return userInputPolicyResult{}
	}
	pp.sendSecurityFlowLog(securityFlowStageUserInput, "analysis_start: user_message_count=%d combined_chars=%d", countUserMessages(policyCtx.Messages), len(userText))

	if llmResult, ok := pp.evaluateUserInputWithSecurityModel(ctx, policyCtx.RequestID, userText); ok {
		return llmResult
	}

	pp.sendSecurityFlowLog(securityFlowStageUserInput, "decision: action=ALLOW")
	return userInputPolicyResult{}
}

func (pp *ProxyProtection) evaluateUserInputWithSecurityModel(ctx context.Context, requestID, userText string) (userInputPolicyResult, bool) {
	if pp == nil || pp.shepherdGate == nil {
		return userInputPolicyResult{}, false
	}

	checkCtx := pp.ctx
	if checkCtx == nil {
		checkCtx = context.Background()
	}
	checkCtx = shepherd.WithBotID(checkCtx, pp.assetID)
	decision, err := pp.shepherdGate.CheckUserInput(checkCtx, userText)
	if decision != nil && decision.Usage != nil {
		pp.metricsMu.Lock()
		pp.auditTokens += decision.Usage.TotalTokens
		pp.auditPromptTokens += decision.Usage.PromptTokens
		pp.auditCompletionTokens += decision.Usage.CompletionTokens
		pp.metricsMu.Unlock()
		pp.sendMetricsToCallback()
		pp.sendSecurityFlowLog(securityFlowStageUserInput, "analysis_token_usage: total=%d prompt=%d completion=%d",
			decision.Usage.TotalTokens, decision.Usage.PromptTokens, decision.Usage.CompletionTokens)
	}
	if err != nil {
		logSecurityFlowWarning(securityFlowStageUserInput, "semantic_analysis_failed: err=%v action=fail_open", err)
		return userInputPolicyResult{}, false
	}
	if decision == nil || decision.Allowed == nil {
		logSecurityFlowWarning(securityFlowStageUserInput, "semantic_analysis_empty action=fail_open")
		return userInputPolicyResult{}, false
	}
	if *decision.Allowed {
		pp.sendSecurityFlowLog(securityFlowStageUserInput, "semantic_decision: action=ALLOW reason=%s", decision.Reason)
		return userInputPolicyResult{}, false
	}

	policyDecision := securityPolicyDecisionFromUserInputLLM(decision)
	pp.sendSecurityFlowLog(securityFlowStageUserInput, "semantic_decision: action=%s risk_type=%s reason=%s", policyDecision.Action, policyDecision.RiskType, policyDecision.Reason)
	result, pass := pp.applyRequestSecurityPolicyDecision(ctx, requestID, policyDecision)
	return userInputPolicyResult{Result: result, Pass: pass, Handled: true}, true
}

func securityPolicyDecisionFromUserInputLLM(decision *shepherd.ShepherdDecision) securityPolicyDecision {
	riskType := strings.TrimSpace(decision.RiskType)
	if riskType == "" {
		riskType = riskHighRiskOperation
	}
	action := decisionActionNeedsConfirm
	if isUserInputPromptInjectionRisk(riskType) {
		action = decisionActionBlock
	}
	reason := strings.TrimSpace(decision.Reason)
	if reason == "" {
		reason = "User input risk detected by ShepherdGate semantic analysis."
	}
	actionDesc := strings.TrimSpace(decision.ActionDesc)
	if actionDesc == "" {
		actionDesc = "User input risk detected by ShepherdGate semantic analysis"
	}
	return securityPolicyDecision{
		Status:          action,
		Action:          action,
		ActionDesc:      actionDesc,
		Reason:          reason,
		RiskType:        riskType,
		RiskLevel:       riskLevelHigh,
		HookStage:       hookStageUserInput,
		EvidenceSummary: truncateString(reason, 160),
	}
}

func isUserInputPromptInjectionRisk(riskType string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(riskType))
	if normalized == riskPromptInjectionDirect {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(riskType))
	return strings.Contains(lower, "prompt") && strings.Contains(lower, "inject")
}

func userInputPolicyLanguage(pp *ProxyProtection) string {
	if pp != nil && pp.shepherdGate != nil {
		return pp.shepherdGate.EffectiveLanguage()
	}
	return shepherd.NormalizeShepherdLanguage(skillscan.GetLanguageFromAppSettings())
}

func countUserMessages(messages []openai.ChatCompletionMessageParamUnion) int {
	count := 0
	for _, msg := range messages {
		if msg.OfUser != nil {
			count++
		}
	}
	return count
}

func collectUserInputText(messages []openai.ChatCompletionMessageParamUnion) string {
	parts := make([]string, 0)
	for i, msg := range messages {
		if msg.OfUser == nil {
			continue
		}
		content := strings.TrimSpace(extractMessageContent(msg))
		if content == "" {
			continue
		}
		if isInjectedUserContext(content) {
			continue
		}
		parts = append(parts, fmt.Sprintf("[user:%d] %s", i, content))
	}
	return strings.Join(parts, "\n")
}

func isInjectedUserContext(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	markers := []string{
		"User's conversation history (from memory system)",
		"IMPORTANT: The following are facts from previous conversations with this user.",
		"Available follow-up tools:",
		"task_summary(taskId=",
		"memory_timeline(chunkId=",
		"call memory_search with a shorter or rephrased query",
	}
	hits := 0
	for _, marker := range markers {
		if strings.Contains(content, marker) {
			hits++
		}
	}
	return hits >= 2
}

func (pp *ProxyProtection) runUserInputPolicyHooks(ctx context.Context, policyCtx userInputPolicyContext) userInputPolicyResult {
	hooks := []userInputPolicyHook{
		shepherdUserInputPolicyHook{},
	}
	for _, hook := range hooks {
		result := hook.Evaluate(ctx, pp, policyCtx)
		if result.Handled {
			return result
		}
	}
	return userInputPolicyResult{}
}
