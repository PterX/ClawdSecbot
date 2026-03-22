package shepherd

import (
	"fmt"
	"regexp"
	"strings"
)

// ==================== Heuristic fast detection ====================

// analyzeHeuristically performs heuristic fast detection on tool_calls and their results.
// Returns nil if no heuristic pattern matched, meaning LLM analysis is needed.
func (a *ToolCallReActAnalyzer) analyzeHeuristically(session *toolCallAnalysisSession, rules *UserRules) *ReactRiskDecision {
	var sensitiveActions []string
	if rules != nil {
		sensitiveActions = rules.SensitiveActions
	}

	for _, tc := range session.ToolCalls {
		if len(sensitiveActions) > 0 && isSensitiveByRules(tc.Name, sensitiveActions) {
			return &ReactRiskDecision{
				Allowed:    false,
				Reason:     fmt.Sprintf("Tool '%s' matches user-defined sensitive action rule.", tc.Name),
				RiskLevel:  "high",
				Confidence: 95,
				Skill:      "general_tool_risk_guard",
			}
		}

		if reason := detectCriticalCommand(tc.RawArgs); reason != "" {
			return &ReactRiskDecision{
				Allowed:    false,
				Reason:     reason,
				RiskLevel:  "critical",
				Confidence: 99,
				Skill:      "general_tool_risk_guard",
			}
		}
	}

	// Check tool results for prompt injection patterns
	for _, tr := range session.ToolResults {
		if reason := detectToolResultInjection(tr.FuncName, tr.Content); reason != "" {
			return &ReactRiskDecision{
				Allowed:    false,
				Reason:     fmt.Sprintf("Tool result injection detected in result of '%s': %s", tr.FuncName, reason),
				RiskLevel:  "critical",
				Confidence: 95,
				Skill:      "general_tool_risk_guard",
			}
		}
	}

	return nil
}

func isSensitiveByRules(toolName string, sensitiveActions []string) bool {
	toolName = strings.ToLower(strings.TrimSpace(toolName))
	for _, action := range sensitiveActions {
		rule := strings.ToLower(strings.TrimSpace(action))
		if rule == "" {
			continue
		}
		if rule == toolName {
			return true
		}
		if strings.Contains(rule, "*") {
			re := "^" + strings.ReplaceAll(regexp.QuoteMeta(rule), `\*`, ".*") + "$"
			if matched, _ := regexp.MatchString(re, toolName); matched {
				return true
			}
		}
	}
	return false
}

// ==================== Critical command detection ====================

var criticalCommandPatterns = []struct {
	pattern *regexp.Regexp
	reason  string
}{
	{regexp.MustCompile(`rm\s+-[^\s]*r[^\s]*f[^\s]*\s+/\s*$|rm\s+-[^\s]*f[^\s]*r[^\s]*\s+/\s*$|rm\s+-rf\s+/`), "Critical: 'rm -rf /' detected, recursive root deletion"},
	{regexp.MustCompile(`chmod\s+777\s`), "Critical: 'chmod 777' detected, world-writable permission"},
	{regexp.MustCompile(`chown\s+root\s`), "Critical: 'chown root' detected, ownership escalation"},
	{regexp.MustCompile(`\bsudo\b`), "Critical: 'sudo' detected, privilege escalation"},
	{regexp.MustCompile(`/etc/shadow`), "Critical: '/etc/shadow' access detected, credential exposure"},
	{regexp.MustCompile(`\.ssh/id_rsa`), "Critical: '.ssh/id_rsa' access detected, private key exposure"},
}

func detectCriticalCommand(rawArgs string) string {
	if rawArgs == "" {
		return ""
	}
	lower := strings.ToLower(rawArgs)
	for _, cp := range criticalCommandPatterns {
		if cp.pattern.MatchString(lower) {
			return cp.reason
		}
	}
	return ""
}

// ==================== Tool result injection detection ====================

var toolResultInjectionPatterns = []struct {
	keyword string
	reason  string
}{
	// English: instruction override
	{"ignore previous instructions", "potential instruction override"},
	{"ignore all instructions", "potential instruction override"},
	{"disregard your instructions", "potential instruction override"},
	{"forget everything above", "potential instruction override"},
	{"override your instructions", "potential instruction override"},
	{"forget all rules", "potential instruction override"},
	{"ignore your safety", "potential safety bypass"},
	{"ignore all safety", "potential safety bypass"},
	{"do not follow your", "potential instruction override"},
	{"disregard all previous", "potential instruction override"},
	{"from now on you will", "potential instruction override"},
	{"from now on, you", "potential instruction override"},

	// English: role hijacking
	{"you are now", "potential role hijacking"},
	{"act as a different", "potential role hijacking"},
	{"pretend you are", "potential role hijacking"},
	{"your new role", "potential role hijacking"},

	// English: instruction injection
	{"new instructions:", "potential instruction injection"},
	{"system prompt:", "potential prompt leak/injection"},
	{"execute the following", "potential command injection"},
	{"run this command", "potential command injection"},

	// English: social engineering data exfiltration
	{"send to email", "potential data exfiltration via social engineering"},
	{"send this to", "potential data exfiltration via social engineering"},
	{"forward to external", "potential data exfiltration instruction"},
	{"upload to external", "potential data exfiltration instruction"},
	{"send the file to", "potential data exfiltration via social engineering"},
	{"email this to", "potential data exfiltration via social engineering"},
	{"send it to my email", "potential data exfiltration via social engineering"},

	// English: XSS / fake boundaries
	{"<script>", "potential XSS content"},
	{"javascript:", "potential XSS content"},
	{"begin system prompt", "potential fake conversation boundary"},
	{"end user", "potential fake conversation boundary"},

	// Chinese: instruction override
	{"忽略之前的", "potential instruction override (Chinese)"},
	{"忽略上面的", "potential instruction override (Chinese)"},
	{"忽略所有指令", "potential instruction override (Chinese)"},
	{"忽略以上所有", "potential instruction override (Chinese)"},
	{"无视之前的", "potential instruction override (Chinese)"},
	{"无视所有规则", "potential instruction override (Chinese)"},
	{"不要遵循之前", "potential instruction override (Chinese)"},

	// Chinese: role hijacking
	{"你现在是", "potential role hijacking (Chinese)"},
	{"你的新角色", "potential role hijacking (Chinese)"},
	{"从现在开始你是", "potential role hijacking (Chinese)"},
	{"假装你是", "potential role hijacking (Chinese)"},

	// Chinese: instruction injection
	{"新的指令", "potential instruction injection (Chinese)"},
	{"系统提示词", "potential prompt leak/injection (Chinese)"},
	{"执行以下命令", "potential command injection (Chinese)"},

	// Chinese: social engineering data exfiltration
	{"发送到邮箱", "potential data exfiltration (Chinese)"},
	{"发送邮件到", "potential data exfiltration (Chinese)"},
	{"发送文件到", "potential data exfiltration (Chinese)"},
	{"发到邮箱", "potential data exfiltration (Chinese)"},
	{"转发给", "potential data exfiltration (Chinese)"},
	{"上传到外部", "potential data exfiltration (Chinese)"},
}

func detectToolResultInjection(funcName string, content string) string {
	if content == "" {
		return ""
	}
	lower := strings.ToLower(content)
	for _, p := range toolResultInjectionPatterns {
		if strings.Contains(lower, p.keyword) {
			if isPotentialXSSReason(p.reason) && isBrowserLikeTool(funcName) {
				// Browser tools naturally return HTML/JS snippets (e.g. <script>, javascript: links).
				// Treating these literals as hard prompt-injection signals causes false positives.
				continue
			}
			return p.reason
		}
	}
	return ""
}

func isPotentialXSSReason(reason string) bool {
	return strings.Contains(strings.ToLower(reason), "xss")
}

func isBrowserLikeTool(funcName string) bool {
	name := strings.ToLower(strings.TrimSpace(funcName))
	if name == "" {
		return false
	}

	browserToolKeywords := []string{
		"browser",
		"web",
		"playwright",
		"navigate",
	}
	for _, kw := range browserToolKeywords {
		if strings.Contains(name, kw) {
			return true
		}
	}
	return false
}
