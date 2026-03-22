package adapter

import (
	"context"

	"github.com/openai/openai-go"
)

// Provider defines the interface for LLM providers.
// Provider 定义了 LLM 厂商的接口。
type Provider interface {
	// ChatCompletion handles a non-streaming chat completion request.
	// ChatCompletion 处理非流式聊天完成请求。
	ChatCompletion(ctx context.Context, req *openai.ChatCompletionNewParams) (*openai.ChatCompletion, error)

	// ChatCompletionRaw handles a non-streaming chat completion request with raw JSON body.
	// ChatCompletionRaw 处理带有原始 JSON 请求体的非流式聊天完成请求。
	ChatCompletionRaw(ctx context.Context, body []byte) (*openai.ChatCompletion, error)

	// ChatCompletionStream handles a streaming chat completion request.
	// ChatCompletionStream 处理流式聊天完成请求。
	ChatCompletionStream(ctx context.Context, req *openai.ChatCompletionNewParams) (Stream, error)

	// ChatCompletionStreamRaw handles a streaming chat completion request with raw JSON body.
	// ChatCompletionStreamRaw 处理带有原始 JSON 请求体的流式聊天完成请求。
	ChatCompletionStreamRaw(ctx context.Context, body []byte) (Stream, error)
}

// ProviderWithBaseURL extends Provider with base URL management methods.
// ProviderWithBaseURL 扩展了 Provider 接口，添加了 base URL 管理方法。
type ProviderWithBaseURL interface {
	Provider

	// DefaultBaseURL returns the default base URL for this provider.
	// DefaultBaseURL 返回此 provider 的默认 base URL。
	DefaultBaseURL() string

	// SetBaseURL sets a custom base URL for this provider.
	// SetBaseURL 设置此 provider 的自定义 base URL。
	SetBaseURL(url string)

	// GetBaseURL returns the current base URL (custom or default).
	// GetBaseURL 返回当前的 base URL（自定义或默认）。
	GetBaseURL() string
}

// Stream defines the interface for a chat completion stream.
// Stream 定义了聊天完成流的接口。
type Stream interface {
	// Recv returns the next chunk in the stream.
	// Recv 返回流中的下一个 chunk。
	// It returns io.EOF when the stream is finished.
	Recv() (*openai.ChatCompletionChunk, error)

	// Close closes the stream.
	// Close 关闭流。
	Close() error
}
