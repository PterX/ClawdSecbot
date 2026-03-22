package adapter

import "testing"

func TestGetModelMaxOutputTokens(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected int
	}{
		// Anthropic 前缀匹配
		{"claude-3-5-sonnet 带版本号", "claude-3-5-sonnet-20241022", 8192},
		{"claude-3-5-sonnet-latest", "claude-3-5-sonnet-latest", 8192},
		{"claude-3-opus 带版本号", "claude-3-opus-20240229", 4096},
		{"claude-3-7-sonnet", "claude-3-7-sonnet-20250219", 16000},
		{"claude-4-sonnet", "claude-4-sonnet-20250514", 16000},
		{"claude-4-opus", "claude-4-opus-20250514", 32000},
		{"claude-3-haiku", "claude-3-haiku-20240307", 4096},
		{"claude-3-5-haiku", "claude-3-5-haiku-20241022", 8192},

		// OpenAI GPT 系列
		{"gpt-4o", "gpt-4o", 16384},
		{"gpt-4o-mini", "gpt-4o-mini", 16384},
		{"gpt-4o-mini 带日期", "gpt-4o-mini-2024-07-18", 16384},
		{"gpt-4-turbo", "gpt-4-turbo-preview", 4096},
		{"gpt-4", "gpt-4-0613", 8192},
		{"gpt-3.5-turbo", "gpt-3.5-turbo-0125", 16384},
		{"gpt-4.1", "gpt-4.1", 32768},
		{"gpt-4.1-mini", "gpt-4.1-mini", 32768},
		{"gpt-4.1-nano", "gpt-4.1-nano", 32768},

		// OpenAI o 系列（冲突测试）
		{"o3-mini 不被 o3 匹配", "o3-mini", 100000},
		{"o3 精确", "o3", 100000},
		{"o1-mini 不被 o1 匹配", "o1-mini", 65536},
		{"o1 精确", "o1", 100000},
		{"o4-mini", "o4-mini", 100000},

		// Google Gemini 系列
		{"gemini-2.5-pro 带版本", "gemini-2.5-pro-preview-0506", 65536},
		{"gemini-2.5-flash", "gemini-2.5-flash-preview-04-17", 65536},
		{"gemini-2.0-flash", "gemini-2.0-flash", 8192},
		{"gemini-2.0-flash-lite", "gemini-2.0-flash-lite", 8192},
		{"gemini-1.5-pro", "gemini-1.5-pro-002", 8192},
		{"gemini-1.5-flash", "gemini-1.5-flash", 8192},

		// DeepSeek 系列
		{"deepseek-chat", "deepseek-chat", 8192},
		{"deepseek-reasoner", "deepseek-reasoner", 16384},
		{"deepseek 通用前缀", "deepseek-v3", 8192},

		// 其他厂商
		{"minimax 模型", "minimax-abab6.5", 8192},
		{"qwen 模型", "qwen-max", 8192},
		{"glm-5 模型", "glm-5", 16384},
		{"glm-4 模型", "glm-4", 8192},
		{"glm 模型", "glm-3", 4096},
		{"kimi-k2.5 模型", "kimi-k2.5", 16384},
		{"kimi-k2 模型", "kimi-k2-thinking", 16384},
		{"magistral 模型", "magistral-medium-latest", 16384},
		{"doubao-seed-2.0 模型", "doubao-seed-2.0-pro", 16384},
		{"doubao-seed-1 模型", "doubao-seed-1-6-250615", 8192},
		{"hunyuan-t1 模型", "hunyuan-t1-latest", 16384},
		{"hunyuan 模型", "hunyuan-turbos-latest", 8192},
		{"ernie 模型", "ernie-4.5-8k", 8192},
		{"qwq 模型", "qwq-plus", 32768},

		// 大写模型名
		{"大写 Claude", "Claude-3-5-Sonnet-20241022", 8192},
		{"大写 MiniMax", "MiniMax-M2", 8192},

		// 边界条件
		{"未知模型返回默认值", "unknown-model-xyz", 8192},
		{"空字符串返回默认值", "", 8192},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetModelMaxOutputTokens(tt.model)
			if got != tt.expected {
				t.Errorf("GetModelMaxOutputTokens(%q) = %d, want %d", tt.model, got, tt.expected)
			}
		})
	}
}
