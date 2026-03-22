package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go_lib/chatmodel-routing/adapter"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/respjson"
	"github.com/openai/openai-go/packages/ssestream"
	"github.com/tidwall/sjson"
)

const defaultBaseURL = "https://api.deepseek.com/chat/completions"

// Provider 实现 DeepSeek 的协议适配。
// 核心职责：响应后处理——从 DeepSeek R1 响应中提取 reasoning_content 字段，
// 保留思维链数据供代理透传。
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

func (p *Provider) ChatCompletion(ctx context.Context, req *openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	return p.ChatCompletionRaw(ctx, body)
}

func (p *Provider) ChatCompletionRaw(ctx context.Context, body []byte) (*openai.ChatCompletion, error) {
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
		return nil, fmt.Errorf("deepseek api error: %d - %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result openai.ChatCompletion
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 手动提取 reasoning_content（DeepSeek R1 的非标准字段）
	var rawMap map[string]interface{}
	if err := json.Unmarshal(respBody, &rawMap); err == nil {
		if choices, ok := rawMap["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if message, ok := choice["message"].(map[string]interface{}); ok {
					if reasoning, ok := message["reasoning_content"].(string); ok {
						if len(result.Choices) > 0 {
							if result.Choices[0].Message.JSON.ExtraFields == nil {
								result.Choices[0].Message.JSON.ExtraFields = make(map[string]respjson.Field)
							}
							quoted := fmt.Sprintf("%q", reasoning)
							result.Choices[0].Message.JSON.ExtraFields["reasoning_content"] = respjson.NewField(quoted)
						}
					}
				}
			}
		}
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
		return nil, fmt.Errorf("deepseek api error: %d - %s", resp.StatusCode, string(respBody))
	}

	decoder := ssestream.NewDecoder(resp)

	return &deepSeekStream{
		decoder: decoder,
		body:    resp.Body,
	}, nil
}

type deepSeekStream struct {
	decoder ssestream.Decoder
	body    io.ReadCloser
}

func (s *deepSeekStream) Recv() (*openai.ChatCompletionChunk, error) {
	if s.decoder.Next() {
		event := s.decoder.Event()

		// 跳过 [DONE] 结束标记
		if string(bytes.TrimSpace(event.Data)) == "[DONE]" {
			return nil, io.EOF
		}

		var chunk openai.ChatCompletionChunk
		if err := json.Unmarshal(event.Data, &chunk); err != nil {
			return nil, err
		}

		// 提取 DeepSeek R1 流式 reasoning_content
		var rawMap map[string]interface{}
		if err := json.Unmarshal(event.Data, &rawMap); err == nil {
			if choices, ok := rawMap["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if delta, ok := choice["delta"].(map[string]interface{}); ok {
						if reasoning, ok := delta["reasoning_content"].(string); ok {
							if len(chunk.Choices) > 0 {
								if chunk.Choices[0].Delta.JSON.ExtraFields == nil {
									chunk.Choices[0].Delta.JSON.ExtraFields = make(map[string]respjson.Field)
								}
								quoted := fmt.Sprintf("%q", reasoning)
								chunk.Choices[0].Delta.JSON.ExtraFields["reasoning_content"] = respjson.NewField(quoted)
							}
						}
					}
				}
			}
		}

		return &chunk, nil
	}

	if err := s.decoder.Err(); err != nil {
		return nil, err
	}

	return nil, io.EOF
}

func (s *deepSeekStream) Close() error {
	return s.body.Close()
}
