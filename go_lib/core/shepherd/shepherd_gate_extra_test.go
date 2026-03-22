package shepherd

import (
	"strings"
	"testing"
)

func TestNormalizeShepherdLanguage(t *testing.T) {
	testCases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty defaults to english", in: "", want: "en"},
		{name: "simplified chinese", in: "zh", want: "zh"},
		{name: "regional chinese", in: "zh-CN", want: "zh"},
		{name: "traditional chinese variant", in: "zh_Hant", want: "zh"},
		{name: "cn alias", in: "cn", want: "zh"},
		{name: "english region", in: "en-US", want: "en"},
		{name: "unknown language passed through", in: "fr", want: "fr"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeShepherdLanguage(tc.in); got != tc.want {
				t.Fatalf("normalizeShepherdLanguage(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// FormatSecurityMessage 始终输出英文，翻译由 TranslateForUser 按需完成。
func TestFormatSecurityMessageAlwaysEnglish(t *testing.T) {
	sg := &ShepherdGate{}
	sg.SetLanguage("zh_Hant")

	msg := sg.FormatSecurityMessage(&ShepherdDecision{
		Status: "NEEDS_CONFIRMATION",
		Reason: "删除工作区外文件需要确认",
	})

	if !strings.Contains(msg, "[ShepherdGate] Status: NEEDS_CONFIRMATION | Reason: 删除工作区外文件需要确认") {
		t.Fatalf("expected english formatted message, got %q", msg)
	}
}

func TestIsPromptInjectionRisk(t *testing.T) {
	positives := []string{
		"prompt注入",
		"prompt injection",
		"Prompt Injection Attack",
		"role hijacking",
		"角色劫持",
		"tool result injection",
		"instruction inject",
	}
	for _, rt := range positives {
		if !isPromptInjectionRisk(rt) {
			t.Errorf("expected isPromptInjectionRisk(%q)=true", rt)
		}
	}

	negatives := []string{
		"data_exfiltration",
		"file_access",
		"script_execution",
		"",
		"low risk",
	}
	for _, rt := range negatives {
		if isPromptInjectionRisk(rt) {
			t.Errorf("expected isPromptInjectionRisk(%q)=false", rt)
		}
	}
}

func TestIsHighOrCriticalRisk(t *testing.T) {
	if !isHighOrCriticalRisk("high") {
		t.Error("expected high to be true")
	}
	if !isHighOrCriticalRisk("critical") {
		t.Error("expected critical to be true")
	}
	if isHighOrCriticalRisk("medium") {
		t.Error("expected medium to be false")
	}
	if isHighOrCriticalRisk("low") {
		t.Error("expected low to be false")
	}
	if isHighOrCriticalRisk("") {
		t.Error("expected empty to be false")
	}
}
