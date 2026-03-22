package shepherd

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestFormatSecurityMessage_AlwaysEnglish(t *testing.T) {
	sg := &ShepherdGate{language: "zh"}

	msg := sg.FormatSecurityMessage(&ShepherdDecision{
		Status: "NEEDS_CONFIRMATION",
		Reason: "script execution requires confirmation",
	})
	// FormatSecurityMessage 始终返回英文，翻译由 TranslateForUser 按需完成
	if !strings.Contains(msg, "Status") {
		t.Fatalf("expected English status label, got: %s", msg)
	}
	if !strings.Contains(msg, "script execution requires confirmation") {
		t.Fatalf("expected original English reason preserved, got: %s", msg)
	}
}

func TestEvaluateRecoveryIntent_ParseAndNormalize(t *testing.T) {
	sg := &ShepherdGate{
		language: "zh",
		chatModel: &stubChatModel{
			generateResp: &schema.Message{
				Content: `{"intent":"confirm","reason":"用户已确认。","usage":{"prompt_tokens":8,"completion_tokens":4,"total_tokens":12}}`,
				Extra: map[string]interface{}{
					"usage": map[string]interface{}{
						"prompt_tokens":     8,
						"completion_tokens": 4,
						"total_tokens":      12,
					},
				},
			},
		},
	}

	got, err := sg.EvaluateRecoveryIntent(context.Background(),
		[]ConversationMessage{
			{Role: "assistant", Content: "[ShepherdGate] 状态: NEEDS_CONFIRMATION"},
			{Role: "user", Content: "确定，继续"},
		},
		[]ToolCallInfo{{Name: "bash_execute", RawArgs: `{"command":"echo hi"}`}},
		"script requires confirmation",
	)
	if err != nil {
		t.Fatalf("EvaluateRecoveryIntent returned error: %v", err)
	}
	if got == nil {
		t.Fatalf("expected non-nil decision")
	}
	if got.Intent != "CONFIRM" {
		t.Fatalf("expected CONFIRM intent, got=%s", got.Intent)
	}
	if got.Usage == nil || got.Usage.TotalTokens != 12 {
		t.Fatalf("expected usage total=12, got=%+v", got.Usage)
	}
}

func TestNormalizeShepherdLanguage_ZhVariants(t *testing.T) {
	cases := []string{"zh", "zh_CN", "zh-CN", "zh-Hans", "ZH_hant", "cn", "Chinese"}
	for _, c := range cases {
		if got := normalizeShepherdLanguage(c); got != "zh" {
			t.Fatalf("expected zh for %q, got %q", c, got)
		}
	}
}
