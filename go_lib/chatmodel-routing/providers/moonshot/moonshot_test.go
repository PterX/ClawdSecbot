package moonshot

import (
	"encoding/json"
	"testing"
)

func TestPreprocessRequestBody_NoMessages(t *testing.T) {
	// 没有 messages 字段，原样返回
	body := []byte(`{"model":"kimi-k2.5","temperature":0.7}`)
	result := preprocessRequestBody(body)
	if string(result) != string(body) {
		t.Errorf("expected unchanged body, got %s", string(result))
	}
}

func TestPreprocessRequestBody_EmptyMessages(t *testing.T) {
	// messages 为空数组，原样返回
	body := []byte(`{"model":"kimi-k2.5","messages":[]}`)
	result := preprocessRequestBody(body)
	if string(result) != string(body) {
		t.Errorf("expected unchanged body, got %s", string(result))
	}
}

func TestPreprocessRequestBody_NoAssistantMessages(t *testing.T) {
	// 只有 user/system 消息，不注入
	body := []byte(`{"messages":[{"role":"system","content":"hi"},{"role":"user","content":"hello"}]}`)
	result := preprocessRequestBody(body)

	var parsed map[string]interface{}
	json.Unmarshal(result, &parsed)
	messages := parsed["messages"].([]interface{})
	for _, msg := range messages {
		m := msg.(map[string]interface{})
		if _, exists := m["reasoning_content"]; exists {
			t.Errorf("unexpected reasoning_content in %s message", m["role"])
		}
	}
}

func TestPreprocessRequestBody_AssistantWithoutReasoningContent(t *testing.T) {
	// assistant 消息缺少 reasoning_content → 注入空字符串
	body := []byte(`{"messages":[{"role":"user","content":"hi"},{"role":"assistant","content":"hello"}]}`)
	result := preprocessRequestBody(body)

	var parsed map[string]interface{}
	json.Unmarshal(result, &parsed)
	messages := parsed["messages"].([]interface{})
	assistantMsg := messages[1].(map[string]interface{})

	rc, exists := assistantMsg["reasoning_content"]
	if !exists {
		t.Fatal("reasoning_content should be injected")
	}
	if rc != "" {
		t.Errorf("expected empty string, got %v", rc)
	}
}

func TestPreprocessRequestBody_AssistantWithReasoningContent(t *testing.T) {
	// assistant 消息已有 reasoning_content → 不重复注入
	body := []byte(`{"messages":[{"role":"assistant","content":"hi","reasoning_content":"thinking..."}]}`)
	result := preprocessRequestBody(body)

	var parsed map[string]interface{}
	json.Unmarshal(result, &parsed)
	messages := parsed["messages"].([]interface{})
	assistantMsg := messages[0].(map[string]interface{})

	rc := assistantMsg["reasoning_content"].(string)
	if rc != "thinking..." {
		t.Errorf("expected original reasoning_content, got %s", rc)
	}
}

func TestPreprocessRequestBody_AssistantWithToolCallsNoReasoningContent(t *testing.T) {
	// assistant 消息有 tool_calls 但无 reasoning_content → 注入
	body := []byte(`{"messages":[
		{"role":"user","content":"query"},
		{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"search","arguments":"{}"}}]},
		{"role":"tool","tool_call_id":"call_1","content":"result"}
	]}`)
	result := preprocessRequestBody(body)

	var parsed map[string]interface{}
	json.Unmarshal(result, &parsed)
	messages := parsed["messages"].([]interface{})
	assistantMsg := messages[1].(map[string]interface{})

	rc, exists := assistantMsg["reasoning_content"]
	if !exists {
		t.Fatal("reasoning_content should be injected for tool_calls message")
	}
	if rc != "" {
		t.Errorf("expected empty string, got %v", rc)
	}

	// tool 消息不应被注入
	toolMsg := messages[2].(map[string]interface{})
	if _, exists := toolMsg["reasoning_content"]; exists {
		t.Error("tool message should not have reasoning_content")
	}
}

func TestPreprocessRequestBody_MultipleAssistantMessagesMixed(t *testing.T) {
	// 多个 assistant 消息，部分有、部分无 reasoning_content
	body := []byte(`{"messages":[
		{"role":"user","content":"q1"},
		{"role":"assistant","content":"a1","reasoning_content":"thought1"},
		{"role":"user","content":"q2"},
		{"role":"assistant","content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"fn","arguments":"{}"}}]},
		{"role":"tool","tool_call_id":"c1","content":"r1"},
		{"role":"assistant","content":"a3"}
	]}`)
	result := preprocessRequestBody(body)

	var parsed map[string]interface{}
	json.Unmarshal(result, &parsed)
	messages := parsed["messages"].([]interface{})

	// index 1: assistant 已有 reasoning_content → 保留原值
	msg1 := messages[1].(map[string]interface{})
	if msg1["reasoning_content"] != "thought1" {
		t.Errorf("index 1: expected 'thought1', got %v", msg1["reasoning_content"])
	}

	// index 3: assistant 有 tool_calls 无 reasoning_content → 注入
	msg3 := messages[3].(map[string]interface{})
	rc3, exists := msg3["reasoning_content"]
	if !exists {
		t.Fatal("index 3: reasoning_content should be injected")
	}
	if rc3 != "" {
		t.Errorf("index 3: expected empty string, got %v", rc3)
	}

	// index 4: tool → 不注入
	msg4 := messages[4].(map[string]interface{})
	if _, exists := msg4["reasoning_content"]; exists {
		t.Error("index 4: tool message should not have reasoning_content")
	}

	// index 5: assistant 无 reasoning_content → 注入
	msg5 := messages[5].(map[string]interface{})
	rc5, exists := msg5["reasoning_content"]
	if !exists {
		t.Fatal("index 5: reasoning_content should be injected")
	}
	if rc5 != "" {
		t.Errorf("index 5: expected empty string, got %v", rc5)
	}
}
