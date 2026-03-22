package xai

import (
	"go_lib/chatmodel-routing/providers/openai"
)

// New creates a new provider for xAI, reusing the OpenAI implementation.
// New 创建一个新的 xAI provider，复用 OpenAI 的实现。
func New(apiKey string) *openai.Provider {
	p := openai.New(apiKey)
	// Set xAI API base URL
	// 设置 xAI API 的基础 URL
	p.SetBaseURL("https://api.x.ai/v1/chat/completions")
	return p
}
