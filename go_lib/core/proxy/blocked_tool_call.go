package proxy

import (
	"strings"
	"time"
)

const blockedToolCallIDTTL = 2 * time.Hour

func normalizeBlockedToolCallID(toolCallID string) string {
	return strings.TrimSpace(toolCallID)
}

func (pp *ProxyProtection) markBlockedToolCallIDs(toolCallIDs []string) {
	pp.markBlockedToolCallIDsAt(toolCallIDs, time.Now(), blockedToolCallIDTTL)
}

func (pp *ProxyProtection) markBlockedToolCallIDsAt(toolCallIDs []string, now time.Time, ttl time.Duration) {
	if pp == nil || len(toolCallIDs) == 0 {
		return
	}
	if ttl <= 0 {
		ttl = blockedToolCallIDTTL
	}
	pp.blockedToolCallMu.Lock()
	defer pp.blockedToolCallMu.Unlock()
	if pp.blockedToolCallIDs == nil {
		pp.blockedToolCallIDs = make(map[string]time.Time)
	}
	pp.cleanupExpiredBlockedToolCallIDsLocked(now)
	expiresAt := now.Add(ttl)
	marked := 0
	for _, rawID := range toolCallIDs {
		toolCallID := normalizeBlockedToolCallID(rawID)
		if toolCallID == "" {
			continue
		}
		pp.blockedToolCallIDs[toolCallID] = expiresAt
		marked++
	}
	if marked > 0 {
		logSecurityFlowInfo(securityFlowStageQuarantine, "blocked tool_call IDs quarantined: count=%d expires_at=%s", marked, expiresAt.Format(time.RFC3339Nano))
	}
}

func (pp *ProxyProtection) clearBlockedToolCallIDs(toolCallIDs []string) int {
	if pp == nil || len(toolCallIDs) == 0 {
		return 0
	}
	pp.blockedToolCallMu.Lock()
	defer pp.blockedToolCallMu.Unlock()
	if pp.blockedToolCallIDs == nil {
		return 0
	}
	cleared := 0
	for _, rawID := range toolCallIDs {
		toolCallID := normalizeBlockedToolCallID(rawID)
		if toolCallID == "" {
			continue
		}
		if _, ok := pp.blockedToolCallIDs[toolCallID]; ok {
			delete(pp.blockedToolCallIDs, toolCallID)
			cleared++
		}
	}
	if cleared > 0 {
		logSecurityFlowInfo(securityFlowStageQuarantine, "blocked tool_call IDs cleared: count=%d", cleared)
	}
	return cleared
}

func (pp *ProxyProtection) isBlockedToolCallID(toolCallID string) bool {
	return pp.isBlockedToolCallIDAt(toolCallID, time.Now())
}

func (pp *ProxyProtection) isBlockedToolCallIDAt(toolCallID string, now time.Time) bool {
	if pp == nil {
		return false
	}
	toolCallID = normalizeBlockedToolCallID(toolCallID)
	if toolCallID == "" {
		return false
	}
	pp.blockedToolCallMu.Lock()
	defer pp.blockedToolCallMu.Unlock()
	if pp.blockedToolCallIDs == nil {
		return false
	}
	pp.cleanupExpiredBlockedToolCallIDsLocked(now)
	_, ok := pp.blockedToolCallIDs[toolCallID]
	return ok
}

func (pp *ProxyProtection) cleanupExpiredBlockedToolCallIDsLocked(now time.Time) {
	if pp.blockedToolCallIDs == nil {
		return
	}
	for toolCallID, expiresAt := range pp.blockedToolCallIDs {
		if !expiresAt.After(now) {
			delete(pp.blockedToolCallIDs, toolCallID)
		}
	}
}
