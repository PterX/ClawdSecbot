package shepherd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestTranslateForUser_EmptyUserMessage(t *testing.T) {
	stub := &stubChatModel{
		generateResp: &schema.Message{Content: "不应被调用"},
	}
	sg := &ShepherdGate{chatModel: stub}

	msg := "[ShepherdGate] Status: NEEDS_CONFIRMATION | Reason: test"
	got := sg.TranslateForUser(context.Background(), msg, "")

	if got != msg {
		t.Errorf("expected original message, got: %s", got)
	}
	if stub.called {
		t.Error("expected LLM not to be called for empty user message")
	}
}

func TestTranslateForUser_EnglishUser(t *testing.T) {
	// 英文用户也会调用 LLM，由 LLM 判断无需翻译直接返回原文
	stub := &stubChatModel{
		generateResp: &schema.Message{Content: "[ShepherdGate] Status: NEEDS_CONFIRMATION | Reason: data exfiltration detected"},
	}
	sg := &ShepherdGate{chatModel: stub}

	msg := "[ShepherdGate] Status: NEEDS_CONFIRMATION | Reason: data exfiltration detected"
	got := sg.TranslateForUser(context.Background(), msg, "Please help me write a function")

	if got != msg {
		t.Errorf("expected message returned as-is by LLM, got: %s", got)
	}
	if !stub.called {
		t.Error("expected LLM to be called even for English user")
	}
}

func TestTranslateForUser_NonEnglishUser(t *testing.T) {
	translated := "[安全守卫] 状态: 需要确认 | 原因: 检测到数据外泄"
	stub := &stubChatModel{
		generateResp: &schema.Message{Content: translated},
	}
	sg := &ShepherdGate{chatModel: stub}

	msg := "[ShepherdGate] Status: NEEDS_CONFIRMATION | Reason: data exfiltration detected"
	got := sg.TranslateForUser(context.Background(), msg, "请帮我写一个函数")

	if got != translated {
		t.Errorf("expected translated message, got: %s", got)
	}
	if !stub.called {
		t.Error("expected LLM to be called for Chinese user")
	}
}

func TestTranslateForUser_LLMError(t *testing.T) {
	stub := &stubChatModel{
		generateErr: errors.New("connection timeout"),
	}
	sg := &ShepherdGate{chatModel: stub}

	msg := "[ShepherdGate] Status: NEEDS_CONFIRMATION | Reason: test"
	got := sg.TranslateForUser(context.Background(), msg, "请帮我写一个函数")

	if got != msg {
		t.Errorf("expected fallback to original message on LLM error, got: %s", got)
	}
}

func TestTranslateForUser_LLMEmptyResponse(t *testing.T) {
	stub := &stubChatModel{
		generateResp: &schema.Message{Content: "   "},
	}
	sg := &ShepherdGate{chatModel: stub}

	msg := "[ShepherdGate] Status: NEEDS_CONFIRMATION | Reason: test"
	got := sg.TranslateForUser(context.Background(), msg, "这是中文消息")

	if got != msg {
		t.Errorf("expected fallback to original message on empty LLM response, got: %s", got)
	}
}

func TestTranslateForUser_NilChatModel(t *testing.T) {
	sg := &ShepherdGate{chatModel: nil}

	msg := "[ShepherdGate] Status: NEEDS_CONFIRMATION | Reason: test"
	got := sg.TranslateForUser(context.Background(), msg, "这是中文消息")

	if got != msg {
		t.Errorf("expected fallback to original message with nil chatModel, got: %s", got)
	}
}

func TestTranslateForUser_TruncatesLongUserMessage(t *testing.T) {
	// 构造超过 200 字符的用户消息，验证不会 panic
	longMsg := strings.Repeat("这是测试", 100) // 400 字符
	translated := "[安全守卫] 状态: 需要确认"
	stub := &stubChatModel{
		generateResp: &schema.Message{Content: translated},
	}
	sg := &ShepherdGate{chatModel: stub}

	msg := "[ShepherdGate] Status: NEEDS_CONFIRMATION | Reason: test"
	got := sg.TranslateForUser(context.Background(), msg, longMsg)

	if got != translated {
		t.Errorf("expected translated message, got: %s", got)
	}
	if !stub.called {
		t.Error("expected LLM to be called")
	}
}
