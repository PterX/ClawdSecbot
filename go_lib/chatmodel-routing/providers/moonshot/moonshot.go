package moonshot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go_lib/chatmodel-routing/adapter"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/ssestream"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const defaultBaseURL = "https://api.moonshot.ai/v1/chat/completions"

// Provider 实现 Moonshot AI (Kimi) 的协议适配。
// 核心职责：请求预处理——为 assistant 消息补全缺失的 reasoning_content 字段，
// 防止 Kimi K2.5 thinking 模式下的 400 校验错误。
// 响应处理与 OpenAI 兼容 provider 一致，由 proxy 层透传原始 JSON。
type Provider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func New(apiKey string) *Provider {
	return &Provider{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{},
	}
}

func (p *Provider) DefaultBaseURL() string {
	return defaultBaseURL
}

func (p *Provider) GetBaseURL() string {
	return p.baseURL
}

func (p *Provider) SetBaseURL(url string) {
	p.baseURL = url
}

func (p *Provider) SetHTTPClient(client *http.Client) {
	p.httpClient = client
}

// 确保实现 ProviderWithBaseURL 接口
var _ adapter.ProviderWithBaseURL = (*Provider)(nil)

// preprocessRequestBody 为 assistant 消息补全缺失的 reasoning_content 字段。
// Moonshot Kimi K2.5 的 thinking 模式要求所有 assistant 消息都包含 reasoning_content，
// 客户端（如 Cursor/Cline）通常会丢弃该非标准字段，导致后续请求被 400 拒绝。
// 此函数在转发前注入空字符串作为安全兜底。
func preprocessRequestBody(body []byte) []byte {
	messages := gjson.GetBytes(body, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return body
	}

	// 收集需要注入 reasoning_content 的 assistant 消息索引
	var patchIndices []int
	messages.ForEach(func(key, value gjson.Result) bool {
		if value.Get("role").String() == "assistant" {
			if !value.Get("reasoning_content").Exists() {
				patchIndices = append(patchIndices, int(key.Int()))
			}
		}
		return true
	})

	if len(patchIndices) == 0 {
		return body
	}

	// 从尾部向头部设置，保证索引稳定
	result := body
	for i := len(patchIndices) - 1; i >= 0; i-- {
		path := fmt.Sprintf("messages.%d.reasoning_content", patchIndices[i])
		patched, err := sjson.SetBytes(result, path, "")
		if err == nil {
			result = patched
		}
	}

	return result
}

func (p *Provider) ChatCompletion(ctx context.Context, req *openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	return p.ChatCompletionRaw(ctx, body)
}

func (p *Provider) ChatCompletionRaw(ctx context.Context, body []byte) (*openai.ChatCompletion, error) {
	// 请求预处理：补全 assistant 消息的 reasoning_content
	body = preprocessRequestBody(body)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("moonshot api error: %d - %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result openai.ChatCompletion
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

func (p *Provider) ChatCompletionStream(ctx context.Context, req *openai.ChatCompletionNewParams) (adapter.Stream, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	return p.ChatCompletionStreamRaw(ctx, body)
}

func (p *Provider) ChatCompletionStreamRaw(ctx context.Context, body []byte) (adapter.Stream, error) {
	// 请求预处理：补全 assistant 消息的 reasoning_content
	body = preprocessRequestBody(body)

	var err error
	body, err = sjson.SetBytes(body, "stream", true)
	if err != nil {
		return nil, fmt.Errorf("failed to set stream=true: %w", err)
	}

	body, err = sjson.SetBytes(body, "stream_options.include_usage", true)
	if err != nil {
		return nil, fmt.Errorf("failed to set stream_options: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("moonshot api error: %d - %s", resp.StatusCode, string(respBody))
	}

	decoder := ssestream.NewDecoder(resp)

	return &moonshotStream{
		decoder: decoder,
		body:    resp.Body,
	}, nil
}

type moonshotStream struct {
	decoder ssestream.Decoder
	body    io.ReadCloser
}

func (s *moonshotStream) Recv() (*openai.ChatCompletionChunk, error) {
	if s.decoder.Next() {
		event := s.decoder.Event()

		if string(bytes.TrimSpace(event.Data)) == "[DONE]" {
			return nil, io.EOF
		}

		var chunk openai.ChatCompletionChunk
		if err := json.Unmarshal(event.Data, &chunk); err != nil {
			return nil, err
		}

		return &chunk, nil
	}

	if err := s.decoder.Err(); err != nil {
		return nil, err
	}

	return nil, io.EOF
}

func (s *moonshotStream) Close() error {
	return s.body.Close()
}
