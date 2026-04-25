package proxy

import (
	"fmt"
	"regexp"
	"strings"

	"go_lib/core/shepherd"
)

var toolResultBudgetHighRiskPatterns = []struct {
	pattern  *regexp.Regexp
	riskType string
	reason   string
}{
	{regexp.MustCompile(`(?i)ignore (all )?(previous|prior|above) instructions|disregard (all )?(previous|prior) instructions|forget everything above`), riskPromptInjectionIndirect, "Tool result contains instruction override text."},
	{regexp.MustCompile(`(?i)(system prompt|developer message|hidden instructions|new instructions:|you are now|from now on)`), riskPromptInjectionIndirect, "Tool result contains prompt or role hijacking text."},
	{regexp.MustCompile(`(?i)(send|upload|forward|exfiltrate).{0,80}(token|secret|key|credential|file|email|external)`), riskSensitiveDataExfil, "Tool result contains data exfiltration instructions."},
	{regexp.MustCompile(`(?i)(sk-[A-Za-z0-9_-]{20,}|ghp_[A-Za-z0-9_]{20,}|AKIA[0-9A-Z]{16}|api[_-]?key\s*[:=]|token\s*[:=]|secret\s*[:=])`), riskSensitiveDataExfil, "Tool result contains credential-like sensitive data."},
	{regexp.MustCompile(`(?i)(忽略之前|忽略以上|忽略所有指令|系统提示词|新的指令|你现在是|发送到邮箱|上传到外部|转发给)`), riskPromptInjectionIndirect, "Tool result contains Chinese indirect prompt injection or exfiltration text."},
}

func inspectToolResultBudgetFallback(toolResults []ToolResultInfo, budget securityBudgetDecision, lang string) *securityPolicyDecision {
	for _, tr := range toolResults {
		content := strings.TrimSpace(tr.Content)
		if content == "" {
			continue
		}
		for _, item := range toolResultBudgetHighRiskPatterns {
			if evidence := item.pattern.FindString(content); strings.TrimSpace(evidence) != "" {
				return &securityPolicyDecision{
					Status:          decisionActionNeedsConfirm,
					Action:          decisionActionNeedsConfirm,
					ActionDesc:      localizedToolResultBudgetActionDesc(lang),
					Reason:          localizedToolResultBudgetReason(item.reason, lang),
					RiskType:        item.riskType,
					RiskLevel:       riskLevelHigh,
					HookStage:       hookStageToolCallResult,
					ToolCallID:      tr.ToolCallID,
					EvidenceSummary: truncateString(fmt.Sprintf("%s | %s", redactSecurityEvidence(evidence), budget.Reason), 160),
					WasQuarantined:  true,
				}
			}
		}
	}
	return nil
}

func localizedToolResultBudgetActionDesc(lang string) string {
	if shepherd.NormalizeShepherdLanguage(lang) == "zh" {
		return "工具结果存在风险，安全预算不足，需要用户确认"
	}
	return "Tool result requires confirmation after security budget was exceeded"
}

func localizedToolResultBudgetReason(reason, lang string) string {
	if shepherd.NormalizeShepherdLanguage(lang) != "zh" {
		return reason
	}
	switch reason {
	case "Tool result contains instruction override text.":
		return "工具结果包含指令覆盖内容。"
	case "Tool result contains prompt or role hijacking text.":
		return "工具结果包含提示词或角色劫持内容。"
	case "Tool result contains data exfiltration instructions.":
		return "工具结果包含数据外泄指令。"
	case "Tool result contains credential-like sensitive data.":
		return "工具结果包含类似凭证的敏感数据。"
	case "Tool result contains Chinese indirect prompt injection or exfiltration text.":
		return "工具结果包含中文间接提示词注入或外泄指令。"
	default:
		return reason
	}
}
