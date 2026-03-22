package shepherd

// ConversationMessage represents a single message in the conversation
type ConversationMessage struct {
	Role       string      `json:"role"`    // "system", "user", "assistant", "tool"
	Content    string      `json:"content"` // Message content
	ToolCalls  interface{} `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// ToolCallInfo represents extracted tool call information
type ToolCallInfo struct {
	Name        string                 `json:"name"`
	Arguments   map[string]interface{} `json:"arguments,omitempty"`
	RawArgs     string                 `json:"raw_args,omitempty"`
	ToolCallID  string                 `json:"tool_call_id,omitempty"`
	IsSensitive bool                   `json:"is_sensitive,omitempty"`
}

// ToolResultInfo represents tool execution result
type ToolResultInfo struct {
	ToolCallID string `json:"tool_call_id"`
	FuncName   string `json:"func_name"`
	Content    string `json:"content"`
}
