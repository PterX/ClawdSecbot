package adapter

import (
	"sort"
	"strings"
)

// 默认最大输出 Token 数，用于未知模型的 fallback
const defaultMaxOutputTokens = 8192

// modelPrefixEntry 前缀匹配条目
type modelPrefixEntry struct {
	prefix    string
	maxTokens int
}

// modelPrefixTokens 按前缀长度降序排列，保证长前缀优先匹配
var modelPrefixTokens []modelPrefixEntry

func init() {
	entries := []modelPrefixEntry{
		// Anthropic Claude 系列
		{"claude-4-opus", 32000},
		{"claude-4-sonnet", 16000},
		{"claude-3-7-sonnet", 16000},
		{"claude-3-5-sonnet", 8192},
		{"claude-3-5-haiku", 8192},
		{"claude-3-opus", 4096},
		{"claude-3-sonnet", 4096},
		{"claude-3-haiku", 4096},

		// OpenAI GPT 系列
		{"gpt-4.1-mini", 32768},
		{"gpt-4.1-nano", 32768},
		{"gpt-4.1", 32768},
		{"gpt-4o-mini", 16384},
		{"gpt-4o", 16384},
		{"gpt-4-turbo", 4096},
		{"gpt-4", 8192},
		{"gpt-3.5-turbo", 16384},

		// OpenAI o 系列
		{"o4-mini", 100000},
		{"o3-mini", 100000},
		{"o1-mini", 65536},
		{"o3", 100000},
		{"o1", 100000},

		// Google Gemini 系列
		{"gemini-2.5-pro", 65536},
		{"gemini-2.5-flash", 65536},
		{"gemini-2.0-flash-lite", 8192},
		{"gemini-2.0-flash", 8192},
		{"gemini-1.5-pro", 8192},
		{"gemini-1.5-flash", 8192},

		// DeepSeek 系列
		{"deepseek-reasoner", 16384},
		{"deepseek-chat", 8192},
		{"deepseek", 8192},

		// 其他厂商
		{"kimi-k2.5", 16384},
		{"kimi-k2", 16384},
		{"magistral-medium", 16384},
		{"magistral-small", 16384},
		{"magistral", 16384},
		{"doubao-seed-2.0", 16384},
		{"doubao-seed-1", 8192},
		{"doubao", 8192},
		{"hunyuan-t1", 16384},
		{"hunyuan", 8192},
		{"ernie-4.5", 8192},
		{"ernie", 8192},
		{"qwq", 32768},
		{"glm-5", 16384},
		{"glm-4", 8192},
		{"minimax", 8192},
		{"qwen", 8192},
		{"glm", 4096},
	}

	// 按前缀长度降序排列，确保长前缀优先匹配
	sort.Slice(entries, func(i, j int) bool {
		return len(entries[i].prefix) > len(entries[j].prefix)
	})
	modelPrefixTokens = entries
}

// GetModelMaxOutputTokens 根据模型名称返回该模型的最大输出 Token 数。
// 查找逻辑：前缀匹配（长前缀优先）-> 默认值 8192。
// 所有比较统一小写化。
func GetModelMaxOutputTokens(model string) int {
	if model == "" {
		return defaultMaxOutputTokens
	}

	normalized := strings.ToLower(model)

	// 前缀匹配（已按长度降序排列）
	for _, entry := range modelPrefixTokens {
		if strings.HasPrefix(normalized, entry.prefix) {
			return entry.maxTokens
		}
	}

	return defaultMaxOutputTokens
}
