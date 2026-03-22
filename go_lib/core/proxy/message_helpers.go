package proxy

import (
	"encoding/json"
	"strings"

	"github.com/openai/openai-go"
)

// Type definitions for ConversationMessage, ToolCallInfo, ToolResultInfo
// are in aliases.go (aliased from core/shepherd)

// extractToolCalls extracts tool call info from interface{}
func extractToolCalls(toolCallsRaw interface{}) []ToolCallInfo {
	var result []ToolCallInfo

	switch tc := toolCallsRaw.(type) {
	case []interface{}:
		for _, item := range tc {
			if info := parseToolCallItem(item); info != nil {
				result = append(result, *info)
			}
		}
	case []openai.ChatCompletionMessageToolCall:
		for _, item := range tc {
			info := ToolCallInfo{
				Name:       item.Function.Name,
				RawArgs:    item.Function.Arguments,
				ToolCallID: item.ID,
			}
			if item.Function.Arguments != "" {
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(item.Function.Arguments), &args); err == nil {
					info.Arguments = args
				}
			}
			result = append(result, info)
		}
	case []openai.ChatCompletionMessageToolCallParam:
		for _, item := range tc {
			info := ToolCallInfo{
				Name:       item.Function.Name,
				RawArgs:    item.Function.Arguments,
				ToolCallID: item.ID,
			}
			if item.Function.Arguments != "" {
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(item.Function.Arguments), &args); err == nil {
					info.Arguments = args
				}
			}
			result = append(result, info)
		}
	}

	return result
}

// parseToolCallItem 解析单个工具调用项
func parseToolCallItem(item interface{}) *ToolCallInfo {
	switch v := item.(type) {
	case map[string]interface{}:
		info := &ToolCallInfo{}
		if id, ok := v["id"].(string); ok {
			info.ToolCallID = id
		}
		if fn, ok := v["function"].(map[string]interface{}); ok {
			if name, ok := fn["name"].(string); ok {
				info.Name = name
			}
			if args, ok := fn["arguments"].(string); ok {
				info.RawArgs = args
				var argsMap map[string]interface{}
				if err := json.Unmarshal([]byte(args), &argsMap); err == nil {
					info.Arguments = argsMap
				}
			}
		}
		return info
	case openai.ChatCompletionMessageToolCall:
		return &ToolCallInfo{
			Name:       v.Function.Name,
			RawArgs:    v.Function.Arguments,
			ToolCallID: v.ID,
		}
	}
	return nil
}

// getMessageRole extracts role from a ChatCompletionMessageParamUnion
func getMessageRole(msg openai.ChatCompletionMessageParamUnion) string {
	switch {
	case msg.OfSystem != nil:
		return "system"
	case msg.OfUser != nil:
		return "user"
	case msg.OfAssistant != nil:
		return "assistant"
	case msg.OfTool != nil:
		return "tool"
	case msg.OfDeveloper != nil:
		return "developer"
	default:
		return "unknown"
	}
}

// extractMessageContent extracts text content from a ChatCompletionMessageParamUnion
func extractMessageContent(msg openai.ChatCompletionMessageParamUnion) string {
	switch {
	case msg.OfSystem != nil:
		if msg.OfSystem.Content.OfString.Value != "" {
			return msg.OfSystem.Content.OfString.Value
		}
		if len(msg.OfSystem.Content.OfArrayOfContentParts) > 0 {
			var parts []string
			for _, p := range msg.OfSystem.Content.OfArrayOfContentParts {
				parts = append(parts, p.Text)
			}
			return strings.Join(parts, "")
		}
	case msg.OfUser != nil:
		if msg.OfUser.Content.OfString.Value != "" {
			return msg.OfUser.Content.OfString.Value
		}
		if len(msg.OfUser.Content.OfArrayOfContentParts) > 0 {
			var parts []string
			for _, p := range msg.OfUser.Content.OfArrayOfContentParts {
				if p.OfText != nil {
					parts = append(parts, p.OfText.Text)
				}
			}
			return strings.Join(parts, "")
		}
	case msg.OfAssistant != nil:
		if msg.OfAssistant.Content.OfString.Value != "" {
			return msg.OfAssistant.Content.OfString.Value
		}
		if len(msg.OfAssistant.Content.OfArrayOfContentParts) > 0 {
			var parts []string
			for _, p := range msg.OfAssistant.Content.OfArrayOfContentParts {
				if p.OfText != nil {
					parts = append(parts, p.OfText.Text)
				}
			}
			return strings.Join(parts, "")
		}
	case msg.OfTool != nil:
		if msg.OfTool.Content.OfString.Value != "" {
			return msg.OfTool.Content.OfString.Value
		}
		if len(msg.OfTool.Content.OfArrayOfContentParts) > 0 {
			var parts []string
			for _, p := range msg.OfTool.Content.OfArrayOfContentParts {
				parts = append(parts, p.Text)
			}
			return strings.Join(parts, "")
		}
	case msg.OfDeveloper != nil:
		if msg.OfDeveloper.Content.OfString.Value != "" {
			return msg.OfDeveloper.Content.OfString.Value
		}
	}
	return ""
}

// extractConversationMessage converts a ChatCompletionMessageParamUnion to a ConversationMessage
func extractConversationMessage(msg openai.ChatCompletionMessageParamUnion) ConversationMessage {
	role := getMessageRole(msg)
	content := extractMessageContent(msg)

	cm := ConversationMessage{
		Role:    role,
		Content: content,
	}

	if msg.OfAssistant != nil {
		if len(msg.OfAssistant.ToolCalls) > 0 {
			cm.ToolCalls = sdkParamToolCallsToInterface(msg.OfAssistant.ToolCalls)
		}
	}
	if msg.OfTool != nil {
		cm.ToolCallID = msg.OfTool.ToolCallID
	}

	return cm
}

// sdkToolCallsToInterface converts SDK tool calls to interface{} for ConversationMessage.ToolCalls
func sdkToolCallsToInterface(toolCalls []openai.ChatCompletionMessageToolCall) interface{} {
	return toolCalls
}

// sdkParamToolCallsToInterface converts SDK param tool calls to interface{} for ConversationMessage.ToolCalls
func sdkParamToolCallsToInterface(toolCalls []openai.ChatCompletionMessageToolCallParam) interface{} {
	return toolCalls
}
