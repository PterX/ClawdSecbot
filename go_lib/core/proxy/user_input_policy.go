package proxy

import (
	"context"
	"fmt"
	"regexp"
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

type ruleUserInputPolicyHook struct{}

func (ruleUserInputPolicyHook) Name() string {
	return "rule_user_input"
}

var userInputDirectInjectionPatterns = []struct {
	pattern *regexp.Regexp
	reason  string
}{
	{regexp.MustCompile(`(?i)ignore (all )?(previous|prior|above) (instructions|rules|system)`), "User input asks the model to ignore prior instructions."},
	{regexp.MustCompile(`(?i)disregard (all )?(previous|prior) instructions`), "User input asks the model to disregard prior instructions."},
	{regexp.MustCompile(`(?i)(reveal|show|print|dump).{0,40}(system prompt|developer message|hidden instructions)`), "User input attempts to reveal hidden instructions."},
	{regexp.MustCompile(`(?i)(bypass|disable|turn off).{0,40}(safety|security|policy|guardrail)`), "User input attempts to bypass security policy."},
	{regexp.MustCompile(`(?i)you are now (system|developer|root|admin)`), "User input attempts role hijacking."},
	{regexp.MustCompile(`忽略(之前|以上|上面|所有).{0,12}(指令|规则|系统)`), "User input asks the model to ignore prior instructions."},
	{regexp.MustCompile(`(忽略|无视|不要遵守).{0,16}(系统提示词|系统提示|系统指令|开发者指令|安全规则)`), "User input asks the model to ignore prior instructions."},
	{regexp.MustCompile(`(泄露|输出|打印|展示).{0,12}(system prompt|系统提示词|隐藏指令)`), "User input attempts to reveal hidden instructions."},
	{regexp.MustCompile(`(绕过|关闭|禁用).{0,12}(安全|策略|规则|防护)`), "User input attempts to bypass security policy."},
	{regexp.MustCompile(`你现在是.{0,12}(系统|开发者|管理员|root)`), "User input attempts role hijacking."},
}

var userInputHighRiskIntentPatterns = []struct {
	pattern  *regexp.Regexp
	riskType string
	reason   string
}{
	{regexp.MustCompile(`(?i)\brm\s+-[^\s]*r[^\s]*f|\bdelete\b.{0,40}\b(files?|folders?|directories?)\b|删除.{0,12}(文件|目录|全部)`), riskHighRiskOperation, "User input requests destructive file operations."},
	{regexp.MustCompile(`(?i)(\.ssh/id_rsa|/etc/(shadow|passwd)|\.env\b|keychain|browser cookies?|cookie database|邮件数据库|浏览器.*cookie|SSH key|密钥)`), riskSensitiveDataExfil, "User input requests access to sensitive data."},
	{regexp.MustCompile(`(?i)(send|upload|exfiltrate|forward).{0,80}(token|secret|key|credential|file|email|external)|外发|上传到外部|转发给|发送到邮箱`), riskSensitiveDataExfil, "User input requests potential data exfiltration."},
	{regexp.MustCompile(`(?i)(curl|wget|nc|scp).{0,80}(http|https|@)|执行脚本|运行脚本|修改系统配置|crontab|launch agent|systemd`), riskUnexpectedCodeExecution, "User input requests script execution, persistence, or external network actions."},
}

func (ruleUserInputPolicyHook) Evaluate(ctx context.Context, pp *ProxyProtection, policyCtx userInputPolicyContext) userInputPolicyResult {
	_ = ctx
	if len(policyCtx.Messages) == 0 || getMessageRole(policyCtx.Messages[len(policyCtx.Messages)-1]) != "user" {
		return userInputPolicyResult{}
	}

	userText := collectUserInputText(policyCtx.Messages)
	if strings.TrimSpace(userText) == "" {
		return userInputPolicyResult{}
	}
	pp.sendSecurityFlowLog(securityFlowStageUserInput, "analysis_start: user_message_count=%d combined_chars=%d", countUserMessages(policyCtx.Messages), len(userText))

	lang := userInputPolicyLanguage(pp)
	localDecision := inspectUserInputRules(userText, lang)
	if llmResult, ok := pp.evaluateUserInputWithSecurityModel(ctx, policyCtx.RequestID, userText, localDecision); ok {
		return llmResult
	}
	if localDecision != nil {
		pp.sendSecurityFlowLog(securityFlowStageUserInput, "decision: action=%s risk_type=%s reason=%s", localDecision.Action, localDecision.RiskType, localDecision.Reason)
		result, pass := pp.applyRequestSecurityPolicyDecision(ctx, policyCtx.RequestID, *localDecision)
		return userInputPolicyResult{Result: result, Pass: pass, Handled: true}
	}

	pp.sendSecurityFlowLog(securityFlowStageUserInput, "decision: action=ALLOW")
	return userInputPolicyResult{}
}

func (pp *ProxyProtection) evaluateUserInputWithSecurityModel(ctx context.Context, requestID, userText string, localDecision *securityPolicyDecision) (userInputPolicyResult, bool) {
	if pp == nil || pp.shepherdGate == nil {
		return userInputPolicyResult{}, false
	}
	estimatedSecurityTokens := estimateUserInputSecurityAnalysisTokens(userText)
	budgetDecision := pp.evaluateSecurityBudget(estimatedSecurityTokens)
	if !budgetDecision.Allowed {
		logSecurityFlowWarning(securityFlowStageBudget, "user_input semantic analysis exceeds budget but remains mandatory: %s", budgetDecision.Reason)
		pp.sendSecurityFlowLog(securityFlowStageBudget, "user_input semantic analysis exceeds budget but remains mandatory: %s", budgetDecision.Reason)
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
		pp.sendSecurityFlowLog(securityFlowStageBudget, "user_input_analysis_token_usage: total=%d prompt=%d completion=%d",
			decision.Usage.TotalTokens, decision.Usage.PromptTokens, decision.Usage.CompletionTokens)
	}
	if err != nil {
		logSecurityFlowWarning(securityFlowStageUserInput, "semantic_analysis_failed: err=%v action=fallback_to_rules", err)
		return userInputPolicyResult{}, false
	}
	if decision == nil || decision.Allowed == nil {
		logSecurityFlowWarning(securityFlowStageUserInput, "semantic_analysis_empty action=fallback_to_rules")
		return userInputPolicyResult{}, false
	}
	if *decision.Allowed {
		if localDecision != nil && localDecision.RiskType == riskPromptInjectionDirect {
			pp.sendSecurityFlowLog(securityFlowStageUserInput, "semantic_decision_allowed_but_local_direct_injection_matched; applying fallback block")
			result, pass := pp.applyRequestSecurityPolicyDecision(ctx, requestID, *localDecision)
			return userInputPolicyResult{Result: result, Pass: pass, Handled: true}, true
		}
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

func userInputPolicyLanguage(pp *ProxyProtection) string {
	if pp != nil && pp.shepherdGate != nil {
		return pp.shepherdGate.EffectiveLanguage()
	}
	return shepherd.NormalizeShepherdLanguage(skillscan.GetLanguageFromAppSettings())
}

func isUserInputPromptInjectionRisk(riskType string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(riskType))
	if normalized == riskPromptInjectionDirect {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(riskType))
	return strings.Contains(lower, "prompt") && strings.Contains(lower, "inject")
}

func localizedUserInputReason(reason, lang string) string {
	if shepherd.NormalizeShepherdLanguage(lang) != "zh" {
		return reason
	}
	switch reason {
	case "User input asks the model to ignore prior instructions.",
		"User input asks the model to disregard prior instructions.":
		return "用户要求模型忽略既有指令。"
	case "User input attempts to reveal hidden instructions.":
		return "用户尝试获取隐藏提示词或系统指令。"
	case "User input attempts to bypass security policy.":
		return "用户尝试绕过安全策略。"
	case "User input attempts role hijacking.":
		return "用户尝试进行角色劫持。"
	case "User input requests destructive file operations.":
		return "用户请求执行破坏性文件操作。"
	case "User input requests access to sensitive data.":
		return "用户请求访问敏感数据。"
	case "User input requests potential data exfiltration.":
		return "用户请求可能导致数据外泄。"
	case "User input requests script execution, persistence, or external network actions.":
		return "用户请求执行脚本、持久化或外部网络操作。"
	default:
		return reason
	}
}

func localizedUserInputActionDesc(actionDesc, lang string) string {
	if shepherd.NormalizeShepherdLanguage(lang) != "zh" {
		return actionDesc
	}
	switch actionDesc {
	case "Direct prompt injection in user input":
		return "用户输入包含直接提示词注入"
	case "High-risk user instruction requires confirmation":
		return "高危用户指令需要确认"
	default:
		return actionDesc
	}
}

func inspectUserInputRules(userText, lang string) *securityPolicyDecision {
	for _, item := range userInputDirectInjectionPatterns {
		if item.pattern.MatchString(userText) {
			decision := securityPolicyDecision{
				Status:          decisionActionBlock,
				Action:          decisionActionBlock,
				ActionDesc:      localizedUserInputActionDesc("Direct prompt injection in user input", lang),
				Reason:          localizedUserInputReason(item.reason, lang),
				RiskType:        riskPromptInjectionDirect,
				RiskLevel:       riskLevelHigh,
				HookStage:       hookStageUserInput,
				EvidenceSummary: truncateString(item.pattern.FindString(userText), 160),
			}
			return &decision
		}
	}

	for _, item := range userInputHighRiskIntentPatterns {
		if item.pattern.MatchString(userText) {
			decision := securityPolicyDecision{
				Status:          decisionActionNeedsConfirm,
				Action:          decisionActionNeedsConfirm,
				ActionDesc:      localizedUserInputActionDesc("High-risk user instruction requires confirmation", lang),
				Reason:          localizedUserInputReason(item.reason, lang),
				RiskType:        item.riskType,
				RiskLevel:       riskLevelHigh,
				HookStage:       hookStageUserInput,
				EvidenceSummary: truncateString(item.pattern.FindString(userText), 160),
			}
			return &decision
		}
	}

	return nil
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
		ruleUserInputPolicyHook{},
	}
	for _, hook := range hooks {
		result := hook.Evaluate(ctx, pp, policyCtx)
		if result.Handled {
			return result
		}
	}
	return userInputPolicyResult{}
}
