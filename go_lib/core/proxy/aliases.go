package proxy

import (
	"go_lib/core/shepherd"
)

// Type aliases for shepherd types used throughout the proxy package.
// This keeps the proxy code clean while properly depending on core/shepherd.
type ConversationMessage = shepherd.ConversationMessage
type ToolCallInfo = shepherd.ToolCallInfo
type ToolResultInfo = shepherd.ToolResultInfo
type ShepherdDecision = shepherd.ShepherdDecision
type RecoveryIntentDecision = shepherd.RecoveryIntentDecision
type UserRules = shepherd.UserRules
type Usage = shepherd.Usage
type ReActSkillRuntimeConfig = shepherd.ReActSkillRuntimeConfig
