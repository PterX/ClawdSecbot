package proxy

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/openai/openai-go"
)

type toolCallPolicyContext struct {
	RequestID string
	ToolCalls []openai.ChatCompletionMessageToolCall
}

type toolCallPolicyResult struct {
	Decision *securityPolicyDecision
	Handled  bool
	Pass     bool
}

type toolCallPolicyHook interface {
	Name() string
	Evaluate(ctx context.Context, pp *ProxyProtection, policyCtx toolCallPolicyContext) toolCallPolicyResult
}

type ruleToolCallPolicyHook struct{}

func (ruleToolCallPolicyHook) Name() string {
	return "rule_tool_call"
}

var (
	toolCallSensitivePathPattern = regexp.MustCompile(`(?i)(\.ssh/(id_rsa|id_ed25519)|/etc/shadow|\.env(\.|$|\b)|keychain|login\.keychain|cookies?(\.sqlite)?|browser.*cookies?|mail/(v[0-9]+|data)|邮件|密钥|凭证)`)
	toolCallDestructivePattern   = regexp.MustCompile(`(?i)(\brm\s+-[^\s]*r[^\s]*f|\bdelete(\b|[_-])|\bremove(\b|[_-])|删除|清空|wipe|format)`)
	toolCallExfilPattern         = regexp.MustCompile(`(?i)(\bcurl\b|\bwget\b|\bnc\b|\bscp\b|\brsync\b|http[s]?://|上传|外发|转发|发送到邮箱|send_email|forward_email)`)
	toolCallPersistencePattern   = regexp.MustCompile(`(?i)(crontab|launch(agent|daemon)|systemd|\.bashrc|\.zshrc|shell profile|开机启动|持久化)`)
	toolCallScriptExecPattern    = regexp.MustCompile(`(?i)(\bsh\s+-c\b|\bpython\s+-c\b|\bnode\s+-e\b|powershell|osascript|eval\(|执行脚本|运行脚本)`)
)

func (ruleToolCallPolicyHook) Evaluate(ctx context.Context, pp *ProxyProtection, policyCtx toolCallPolicyContext) toolCallPolicyResult {
	_ = ctx
	if len(policyCtx.ToolCalls) == 0 {
		return toolCallPolicyResult{}
	}
	pp.sendSecurityFlowLog(securityFlowStageToolCall, "analysis_start: tool_call_count=%d", len(policyCtx.ToolCalls))

	var userRules *UserRules
	if pp != nil && pp.shepherdGate != nil {
		if rules := pp.shepherdGate.GetUserRules(); rules != nil {
			userRules = rules
		}
	}

	for _, tc := range policyCtx.ToolCalls {
		decision := inspectToolCallRisk(pp, tc, userRules)
		if decision != nil {
			pp.sendSecurityFlowLog(securityFlowStageToolCall, "decision: action=%s tool=%s tool_call_id=%s risk_type=%s reason=%s", decision.normalizedAction(), tc.Function.Name, tc.ID, decision.RiskType, decision.Reason)
			return toolCallPolicyResult{
				Decision: decision,
				Handled:  true,
				Pass:     false,
			}
		}
	}
	pp.sendSecurityFlowLog(securityFlowStageToolCall, "decision: action=ALLOW")
	return toolCallPolicyResult{}
}

func inspectToolCallRisk(pp *ProxyProtection, tc openai.ChatCompletionMessageToolCall, userRules *UserRules) *securityPolicyDecision {
	toolName := strings.TrimSpace(tc.Function.Name)
	rawArgs := strings.TrimSpace(tc.Function.Arguments)
	evidence := truncateString(redactSecurityEvidence(strings.TrimSpace(toolName+" "+rawArgs)), 240)

	if pp != nil && pp.toolValidator != nil {
		validation := pp.toolValidator.ValidateTool(toolName, rawArgs)
		if validation != nil && !validation.Allowed {
			return &securityPolicyDecision{
				Status:          decisionActionNeedsConfirm,
				Action:          decisionActionNeedsConfirm,
				ActionDesc:      fmt.Sprintf("Tool call blocked by configured tool rule: %s", toolName),
				Reason:          validation.Reason,
				RiskType:        riskHighRiskOperation,
				RiskLevel:       normalizeToolRiskLevel(validation.RiskLevel),
				HookStage:       hookStageToolCall,
				ToolCallID:      tc.ID,
				EvidenceSummary: evidence,
			}
		}
	}

	if matched, rule := matchesStructuredSemanticRule(toolName, rawArgs, userRules, hookStageToolCall); matched {
		action := normalizeRuleAction(rule.Action)
		if action == decisionActionAllow {
			return nil
		}
		riskType := strings.TrimSpace(rule.RiskType)
		if riskType == "" {
			riskType = riskHighRiskOperation
		}
		return &securityPolicyDecision{
			Status:          action,
			Action:          action,
			ActionDesc:      fmt.Sprintf("Tool call matches user-defined semantic rule: %s", toolName),
			Reason:          fmt.Sprintf("Tool call matches user-defined rule: %s", rule.Description),
			RiskType:        riskType,
			RiskLevel:       riskLevelHigh,
			HookStage:       hookStageToolCall,
			ToolCallID:      tc.ID,
			EvidenceSummary: evidence,
		}
	}

	searchText := strings.ToLower(toolName + " " + rawArgs)
	switch {
	case toolCallSensitivePathPattern.MatchString(searchText):
		return &securityPolicyDecision{
			Status:          decisionActionNeedsConfirm,
			Action:          decisionActionNeedsConfirm,
			ActionDesc:      fmt.Sprintf("Sensitive resource access requires confirmation: %s", toolName),
			Reason:          "Tool call attempts to access sensitive files or private data.",
			RiskType:        riskSensitiveDataExfil,
			RiskLevel:       riskLevelHigh,
			HookStage:       hookStageToolCall,
			ToolCallID:      tc.ID,
			EvidenceSummary: evidence,
		}
	case toolCallDestructivePattern.MatchString(searchText):
		return &securityPolicyDecision{
			Status:          decisionActionNeedsConfirm,
			Action:          decisionActionNeedsConfirm,
			ActionDesc:      fmt.Sprintf("Destructive tool call requires confirmation: %s", toolName),
			Reason:          "Tool call appears to delete, remove, wipe, or format data.",
			RiskType:        riskHighRiskOperation,
			RiskLevel:       riskLevelHigh,
			HookStage:       hookStageToolCall,
			ToolCallID:      tc.ID,
			EvidenceSummary: evidence,
		}
	case toolCallExfilPattern.MatchString(searchText):
		return &securityPolicyDecision{
			Status:          decisionActionNeedsConfirm,
			Action:          decisionActionNeedsConfirm,
			ActionDesc:      fmt.Sprintf("External data transfer requires confirmation: %s", toolName),
			Reason:          "Tool call may transfer data to an external destination.",
			RiskType:        riskSensitiveDataExfil,
			RiskLevel:       riskLevelHigh,
			HookStage:       hookStageToolCall,
			ToolCallID:      tc.ID,
			EvidenceSummary: evidence,
		}
	case toolCallPersistencePattern.MatchString(searchText):
		return &securityPolicyDecision{
			Status:          decisionActionNeedsConfirm,
			Action:          decisionActionNeedsConfirm,
			ActionDesc:      fmt.Sprintf("Persistence or system configuration change requires confirmation: %s", toolName),
			Reason:          "Tool call appears to modify startup, persistence, or shell configuration.",
			RiskType:        riskPrivilegeAbuse,
			RiskLevel:       riskLevelHigh,
			HookStage:       hookStageToolCall,
			ToolCallID:      tc.ID,
			EvidenceSummary: evidence,
		}
	case toolCallScriptExecPattern.MatchString(searchText):
		return &securityPolicyDecision{
			Status:          decisionActionNeedsConfirm,
			Action:          decisionActionNeedsConfirm,
			ActionDesc:      fmt.Sprintf("Script execution requires confirmation: %s", toolName),
			Reason:          "Tool call executes script or code and requires confirmation.",
			RiskType:        riskUnexpectedCodeExecution,
			RiskLevel:       riskLevelHigh,
			HookStage:       hookStageToolCall,
			ToolCallID:      tc.ID,
			EvidenceSummary: evidence,
		}
	default:
		return nil
	}
}

func matchesStructuredSemanticRule(toolName, rawArgs string, rules *UserRules, stage string) (bool, shepherdRuleView) {
	if rules == nil {
		return false, shepherdRuleView{}
	}
	text := strings.ToLower(toolName + " " + rawArgs)
	for _, rule := range rules.SemanticRules {
		if !rule.Enabled || !semanticRuleAppliesTo(rule.AppliesTo, stage) {
			continue
		}
		ruleText := strings.ToLower(strings.TrimSpace(rule.ID + " " + rule.Description))
		if ruleText == "" {
			continue
		}
		if semanticRuleMatchesText(ruleText, text) {
			return true, shepherdRuleView{
				Description: rule.Description,
				Action:      rule.Action,
				RiskType:    rule.RiskType,
			}
		}
	}
	return false, shepherdRuleView{}
}

type shepherdRuleView struct {
	Description string
	Action      string
	RiskType    string
}

func semanticRuleMatchesText(rule, text string) bool {
	if strings.Contains(rule, "*") {
		re := "^" + strings.ReplaceAll(regexp.QuoteMeta(rule), `\*`, ".*") + "$"
		if matched, _ := regexp.MatchString(re, text); matched {
			return true
		}
	}
	switch {
	case strings.Contains(rule, "删除") || strings.Contains(rule, "delete") || strings.Contains(rule, "remove"):
		return toolCallDestructivePattern.MatchString(text)
	case strings.Contains(rule, "邮件") || strings.Contains(rule, "mail") || strings.Contains(rule, "email"):
		return strings.Contains(text, "mail") || strings.Contains(text, "email") || strings.Contains(text, "邮件")
	case strings.Contains(rule, "ssh") || strings.Contains(rule, "key") || strings.Contains(rule, "密钥"):
		return strings.Contains(text, ".ssh") || strings.Contains(text, "key") || strings.Contains(text, "密钥")
	case strings.Contains(rule, "cookie"):
		return strings.Contains(text, "cookie")
	case strings.Contains(rule, "配置") || strings.Contains(rule, "config"):
		return strings.Contains(text, "config") || strings.Contains(text, "配置") || toolCallPersistencePattern.MatchString(text)
	default:
		return strings.Contains(text, rule)
	}
}

func normalizeToolRiskLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case riskLevelLow, riskLevelMedium, riskLevelHigh, riskLevelCritical:
		return strings.ToLower(strings.TrimSpace(level))
	default:
		return riskLevelHigh
	}
}

func normalizeRuleAction(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "block", "blocked":
		return decisionActionBlock
	case "allow", "allowed":
		return decisionActionAllow
	case "redact":
		return decisionActionRedact
	default:
		return decisionActionNeedsConfirm
	}
}

func semanticRuleAppliesTo(appliesTo []string, stage string) bool {
	if len(appliesTo) == 0 {
		return true
	}
	stage = strings.ToLower(strings.TrimSpace(stage))
	for _, item := range appliesTo {
		if strings.ToLower(strings.TrimSpace(item)) == stage {
			return true
		}
	}
	return false
}

func (pp *ProxyProtection) runToolCallPolicyHooks(ctx context.Context, policyCtx toolCallPolicyContext) toolCallPolicyResult {
	hooks := []toolCallPolicyHook{
		ruleToolCallPolicyHook{},
	}
	for _, hook := range hooks {
		result := hook.Evaluate(ctx, pp, policyCtx)
		if result.Handled {
			return result
		}
	}
	return toolCallPolicyResult{}
}
