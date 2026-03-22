package ollama

import (
	"strings"

	"go_lib/chatmodel-routing/adapter"
	openaiProvider "go_lib/chatmodel-routing/providers/openai"
)

const defaultBaseURL = "http://localhost:11434/v1"

// Provider implements the adapter.Provider interface for Ollama.
// Ollama exposes an OpenAI-compatible API at {base_url}/v1/chat/completions,
// so this provider wraps the OpenAI provider with proper URL construction.
// Also works for LM Studio and other local inference servers with the same API layout.
type Provider struct {
	*openaiProvider.Provider
	ollamaBaseURL string
}

// New creates a new Ollama provider.
// apiKey is optional — Ollama typically does not require authentication.
func New(apiKey string) *Provider {
	p := openaiProvider.New(apiKey)
	provider := &Provider{
		Provider:      p,
		ollamaBaseURL: defaultBaseURL,
	}
	p.SetBaseURL(buildEndpointURL(defaultBaseURL))
	return provider
}

// Ensure Provider implements ProviderWithBaseURL interface.
var _ adapter.ProviderWithBaseURL = (*Provider)(nil)

// DefaultBaseURL returns the default Ollama server URL.
func (p *Provider) DefaultBaseURL() string {
	return defaultBaseURL
}

// GetBaseURL returns the current Ollama server URL (not the full endpoint path).
func (p *Provider) GetBaseURL() string {
	return p.ollamaBaseURL
}

// SetBaseURL sets the Ollama server URL directly without any processing.
// The URL from the database is used as-is; the default baseURL in provider
// config is only for pre-filling the UI form.
func (p *Provider) SetBaseURL(url string) {
	p.ollamaBaseURL = url
	p.Provider.SetBaseURL(url)
}

// buildEndpointURL constructs the full OpenAI-compatible chat completions
// endpoint from an Ollama server base URL.
//
// Examples:
//
//	"http://localhost:11434"                     → "http://localhost:11434/v1/chat/completions"
//	"http://localhost:11434/v1"                  → "http://localhost:11434/v1/chat/completions"
//	"http://localhost:11434/v1/chat/completions" → "http://localhost:11434/v1/chat/completions" (unchanged)
func buildEndpointURL(baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/")

	// Already a full endpoint URL — use as-is
	if strings.HasSuffix(baseURL, "/chat/completions") {
		return baseURL
	}

	// Append /v1 if not present
	if !strings.Contains(baseURL, "/v1") {
		baseURL += "/v1"
	}

	return baseURL + "/chat/completions"
}
