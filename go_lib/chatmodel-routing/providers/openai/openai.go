package openai

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
	"github.com/tidwall/sjson"
)

const defaultBaseURL = "https://api.openai.com/v1/chat/completions"

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

// DefaultBaseURL returns the default base URL for OpenAI.
func (p *Provider) DefaultBaseURL() string {
	return defaultBaseURL
}

// GetBaseURL returns the current base URL.
func (p *Provider) GetBaseURL() string {
	return p.baseURL
}

func (p *Provider) SetBaseURL(url string) {
	p.baseURL = url
}

func (p *Provider) SetHTTPClient(client *http.Client) {
	p.httpClient = client
}

// Ensure Provider implements ProviderWithBaseURL interface.
var _ adapter.ProviderWithBaseURL = (*Provider)(nil)

func (p *Provider) ChatCompletion(ctx context.Context, req *openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai api error: %d - %s", resp.StatusCode, string(respBody))
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
		return nil, fmt.Errorf("openai api error: %d - %s", resp.StatusCode, string(respBody))
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

	// Set stream=true using sjson to ensure it's set regardless of struct omitzero
	body, err = sjson.SetBytes(body, "stream", true)
	if err != nil {
		return nil, fmt.Errorf("failed to set stream=true: %w", err)
	}

	// Set stream_options={"include_usage": true} to get token usage in stream
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
		resp.Body.Close()
		return nil, fmt.Errorf("openai api error: %d", resp.StatusCode)
	}

	decoder := ssestream.NewDecoder(resp)

	return &openaiStream{
		decoder: decoder,
		body:    resp.Body,
	}, nil
}

func (p *Provider) ChatCompletionStreamRaw(ctx context.Context, body []byte) (adapter.Stream, error) {
	var err error
	// Set stream=true using sjson to ensure it's set regardless of struct omitzero
	body, err = sjson.SetBytes(body, "stream", true)
	if err != nil {
		return nil, fmt.Errorf("failed to set stream=true: %w", err)
	}

	// Set stream_options={"include_usage": true} to get token usage in stream
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
		return nil, fmt.Errorf("openai api error: %d - %s", resp.StatusCode, string(respBody))
	}

	decoder := ssestream.NewDecoder(resp)

	return &openaiStream{
		decoder: decoder,
		body:    resp.Body,
	}, nil
}

type openaiStream struct {
	decoder ssestream.Decoder
	body    io.ReadCloser
}

func (s *openaiStream) Recv() (*openai.ChatCompletionChunk, error) {
	if s.decoder.Next() {
		event := s.decoder.Event()
		// event is sse.Event.
		// We need to unmarshal event.Data into ChatCompletionChunk

		// Skip [DONE]
		if string(bytes.TrimSpace(event.Data)) == "[DONE]" {
			return nil, io.EOF
		}

		var chunk openai.ChatCompletionChunk
		if err := json.Unmarshal(event.Data, &chunk); err != nil {
			return nil, err
		}

		// OpenAI standard fields are already handled by Unmarshal.
		// If we need to capture extra fields or debug raw data, we can do it here.
		// Currently, we just return the standard chunk.

		return &chunk, nil
	}

	if err := s.decoder.Err(); err != nil {
		return nil, err
	}

	return nil, io.EOF
}

func (s *openaiStream) Close() error {
	return s.body.Close()
}
