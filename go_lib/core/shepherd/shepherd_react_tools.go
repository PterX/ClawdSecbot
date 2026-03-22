package shepherd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go_lib/core/logging"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ==================== get_last_user_message ====================

type guardLastUserMessageTool struct {
	sessions *analysisSessionStore
}

// NewGuardLastUserMessageTool creates the get_last_user_message tool.
func NewGuardLastUserMessageTool(sessions *analysisSessionStore) tool.BaseTool {
	return &guardLastUserMessageTool{sessions: sessions}
}

func (t *guardLastUserMessageTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_last_user_message",
		Desc: "Get the latest user message from session context.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"session_id": {Type: schema.String, Required: true, Desc: "Analysis session id"},
		}),
	}, nil
}

func (t *guardLastUserMessageTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	traceGuard(args.SessionID, "Tool:get_last_user_message", "invoked")
	session, ok := t.sessions.Get(args.SessionID)
	if !ok {
		return "", fmt.Errorf("session not found")
	}

	for i := len(session.Context) - 1; i >= 0; i-- {
		msg := session.Context[i]
		if msg.Role == "user" {
			b, _ := json.Marshal(map[string]interface{}{
				"index":   i,
				"role":    msg.Role,
				"content": msg.Content,
			})
			traceGuard(args.SessionID, "Tool:get_last_user_message", "found index=%d content=%s", i, shortenForLog(msg.Content, 180))
			return string(b), nil
		}
	}

	if session.LastUserMessage != "" {
		b, _ := json.Marshal(map[string]interface{}{
			"index":   -1,
			"role":    "user",
			"content": session.LastUserMessage,
			"source":  "cached",
		})
		traceGuard(args.SessionID, "Tool:get_last_user_message", "using cached lastUserMessage content=%s", shortenForLog(session.LastUserMessage, 180))
		return string(b), nil
	}

	traceGuard(args.SessionID, "Tool:get_last_user_message", "no user message found")
	return `{"index":-1,"role":"user","content":""}`, nil
}

// ==================== search_context_messages ====================

type guardSearchContextTool struct {
	sessions *analysisSessionStore
}

// NewGuardSearchContextTool creates the search_context_messages tool.
func NewGuardSearchContextTool(sessions *analysisSessionStore) tool.BaseTool {
	return &guardSearchContextTool{sessions: sessions}
}

func (t *guardSearchContextTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "search_context_messages",
		Desc: "Search context messages by keyword.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"session_id": {Type: schema.String, Required: true, Desc: "Analysis session id"},
			"keyword":    {Type: schema.String, Required: true, Desc: "Keyword to search"},
			"limit":      {Type: schema.Integer, Required: false, Desc: "Maximum number of matches"},
		}),
	}, nil
}

func (t *guardSearchContextTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		SessionID string `json:"session_id"`
		Keyword   string `json:"keyword"`
		Limit     int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	traceGuard(args.SessionID, "Tool:search_context", "invoked keyword=%s limit=%d", args.Keyword, args.Limit)
	session, ok := t.sessions.Get(args.SessionID)
	if !ok {
		return "", fmt.Errorf("session not found")
	}

	keyword := strings.ToLower(strings.TrimSpace(args.Keyword))
	if keyword == "" {
		return `[]`, nil
	}
	limit := normalizeToolLimit(args.Limit, 8)

	var matches []map[string]interface{}
	for i, msg := range session.Context {
		if strings.Contains(strings.ToLower(msg.Content), keyword) {
			matches = append(matches, map[string]interface{}{
				"index":   i,
				"role":    msg.Role,
				"content": msg.Content,
			})
			if len(matches) >= limit {
				break
			}
		}
	}
	b, _ := json.Marshal(matches)
	traceGuard(args.SessionID, "Tool:search_context", "matches=%d", len(matches))
	return string(b), nil
}

// ==================== get_recent_messages ====================

type guardRecentMessagesTool struct {
	sessions *analysisSessionStore
}

// NewGuardRecentMessagesTool creates the get_recent_messages tool.
func NewGuardRecentMessagesTool(sessions *analysisSessionStore) tool.BaseTool {
	return &guardRecentMessagesTool{sessions: sessions}
}

func (t *guardRecentMessagesTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_recent_messages",
		Desc: "Get recent context messages in chronological order.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"session_id": {Type: schema.String, Required: true, Desc: "Analysis session id"},
			"limit":      {Type: schema.Integer, Required: false, Desc: "How many messages to return"},
		}),
	}, nil
}

func (t *guardRecentMessagesTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		SessionID string `json:"session_id"`
		Limit     int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	traceGuard(args.SessionID, "Tool:get_recent_messages", "invoked limit=%d", args.Limit)
	session, ok := t.sessions.Get(args.SessionID)
	if !ok {
		return "", fmt.Errorf("session not found")
	}

	limit := normalizeToolLimit(args.Limit, 12)
	start := 0
	if len(session.Context) > limit {
		start = len(session.Context) - limit
	}
	b, _ := json.Marshal(session.Context[start:])
	traceGuard(args.SessionID, "Tool:get_recent_messages", "returned messages=%d", len(session.Context[start:]))
	return string(b), nil
}

// ==================== get_recent_tool_calls ====================

type guardRecentToolCallsTool struct {
	sessions *analysisSessionStore
}

// NewGuardRecentToolCallsTool creates the get_recent_tool_calls tool.
func NewGuardRecentToolCallsTool(sessions *analysisSessionStore) tool.BaseTool {
	return &guardRecentToolCallsTool{sessions: sessions}
}

func (t *guardRecentToolCallsTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_recent_tool_calls",
		Desc: "Get tool calls and their execution results of current analysis request.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"session_id": {Type: schema.String, Required: true, Desc: "Analysis session id"},
			"limit":      {Type: schema.Integer, Required: false, Desc: "Maximum count"},
		}),
	}, nil
}

func (t *guardRecentToolCallsTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		SessionID string `json:"session_id"`
		Limit     int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	traceGuard(args.SessionID, "Tool:get_recent_tool_calls", "invoked limit=%d", args.Limit)
	session, ok := t.sessions.Get(args.SessionID)
	if !ok {
		return "", fmt.Errorf("session not found")
	}

	limit := normalizeToolLimit(args.Limit, 8)
	start := 0
	if len(session.ToolCalls) > limit {
		start = len(session.ToolCalls) - limit
	}

	resultMap := make(map[string]string)
	for _, tr := range session.ToolResults {
		content := tr.Content
		if len(content) > 2000 {
			content = content[:2000] + "...(truncated)"
		}
		resultMap[tr.ToolCallID] = content
	}

	type toolCallWithResult struct {
		Name            string                 `json:"name"`
		Arguments       map[string]interface{} `json:"arguments,omitempty"`
		RawArgs         string                 `json:"raw_args,omitempty"`
		ToolCallID      string                 `json:"tool_call_id,omitempty"`
		IsSensitive     bool                   `json:"is_sensitive,omitempty"`
		Result          string                 `json:"result,omitempty"`
		ResultTruncated bool                   `json:"result_truncated,omitempty"`
	}

	selected := session.ToolCalls[start:]
	items := make([]toolCallWithResult, 0, len(selected))
	for _, tc := range selected {
		item := toolCallWithResult{
			Name:        tc.Name,
			Arguments:   tc.Arguments,
			RawArgs:     tc.RawArgs,
			ToolCallID:  tc.ToolCallID,
			IsSensitive: tc.IsSensitive,
		}
		if result, ok := resultMap[tc.ToolCallID]; ok {
			item.Result = result
			item.ResultTruncated = len(result) >= 2000
		}
		items = append(items, item)
	}

	b, _ := json.Marshal(items)
	traceGuard(args.SessionID, "Tool:get_recent_tool_calls", "returned tool_calls=%d (with results)", len(items))
	return string(b), nil
}

// ==================== Tool utility functions ====================

func normalizeToolLimit(limit int, def int) int {
	if limit <= 0 {
		return def
	}
	if limit > 50 {
		return 50
	}
	return limit
}

// createGuardValidateCommand creates the command security validation function.
func createGuardValidateCommand() func(string) error {
	whitelist := map[string]struct{}{
		"cat":     {},
		"head":    {},
		"tail":    {},
		"grep":    {},
		"sed":     {},
		"awk":     {},
		"wc":      {},
		"ls":      {},
		"file":    {},
		"strings": {},
		"echo":    {},
	}

	return func(command string) error {
		command = strings.TrimSpace(command)
		if command == "" {
			return fmt.Errorf("command is required")
		}

		for _, op := range []string{"|", ">", "<", ";", "&&", "||", "`", "$("} {
			if strings.Contains(command, op) {
				logging.ShepherdGateWarning("[ShepherdGate][ValidateCommand] blocked: forbidden operator '%s' in: %s", op, command)
				return fmt.Errorf("command contains forbidden shell operator '%s'", op)
			}
		}

		fields := strings.Fields(command)
		if len(fields) == 0 {
			return fmt.Errorf("empty command")
		}
		baseCmd := fields[0]
		if _, ok := whitelist[baseCmd]; !ok {
			logging.ShepherdGateWarning("[ShepherdGate][ValidateCommand] rejected non-whitelisted command: %s", baseCmd)
			return fmt.Errorf("command '%s' is not in whitelist", baseCmd)
		}

		return nil
	}
}
