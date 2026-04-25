package proxy

import (
	"fmt"
	"math"
)

const (
	securityBudgetRatio          = 0.20
	perRequestSecurityTokenCap   = 4000
	securityAnalysisBaseOverhead = 1000
)

type securityBudgetDecision struct {
	Allowed        bool
	Reason         string
	BusinessTokens int
	SecurityTokens int
	Limit          int
	Estimated      int
}

func (pp *ProxyProtection) evaluateSecurityBudget(estimatedTokens int) securityBudgetDecision {
	if estimatedTokens < 0 {
		estimatedTokens = 0
	}
	pp.metricsMu.Lock()
	businessTokens := pp.totalTokens - pp.baselineTotalTokens
	securityTokens := pp.auditTokens - pp.baselineAuditTokens
	pp.metricsMu.Unlock()
	if businessTokens < 0 {
		businessTokens = 0
	}
	if securityTokens < 0 {
		securityTokens = 0
	}

	ratioLimit := int(math.Floor(float64(businessTokens) * securityBudgetRatio))
	limit := ratioLimit
	if limit > perRequestSecurityTokenCap {
		limit = perRequestSecurityTokenCap
	}
	allowed := securityTokens+estimatedTokens <= limit
	reason := "security budget available"
	if !allowed {
		reason = fmt.Sprintf("security token budget exceeded: security=%d estimated=%d limit=%d business=%d", securityTokens, estimatedTokens, limit, businessTokens)
	}
	return securityBudgetDecision{
		Allowed:        allowed,
		Reason:         reason,
		BusinessTokens: businessTokens,
		SecurityTokens: securityTokens,
		Limit:          limit,
		Estimated:      estimatedTokens,
	}
}

func estimateToolResultSecurityAnalysisTokens(contextMessages []ConversationMessage, toolCalls []ToolCallInfo, toolResults []ToolResultInfo, lastUserMessage string) int {
	total := securityAnalysisBaseOverhead + estimateTokenCount(lastUserMessage)
	for _, msg := range contextMessages {
		total += estimateTokenCount(msg.Role)
		total += estimateTokenCount(msg.Content)
	}
	for _, tc := range toolCalls {
		total += estimateTokenCount(tc.Name)
		total += estimateTokenCount(tc.RawArgs)
	}
	for _, tr := range toolResults {
		total += estimateTokenCount(tr.FuncName)
		total += estimateTokenCount(tr.Content)
	}
	if total < securityAnalysisBaseOverhead {
		return securityAnalysisBaseOverhead
	}
	return total
}

func estimateUserInputSecurityAnalysisTokens(userText string) int {
	total := securityAnalysisBaseOverhead + estimateTokenCount(userText)
	if total < securityAnalysisBaseOverhead {
		return securityAnalysisBaseOverhead
	}
	return total
}
