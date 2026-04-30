package ollama

import "testing"

func TestBuildEndpointURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    string
	}{
		{
			name:    "server root",
			baseURL: "http://localhost:11434",
			want:    "http://localhost:11434/v1/chat/completions",
		},
		{
			name:    "v1 root",
			baseURL: "http://localhost:11434/v1",
			want:    "http://localhost:11434/v1/chat/completions",
		},
		{
			name:    "full endpoint",
			baseURL: "http://localhost:11434/v1/chat/completions",
			want:    "http://localhost:11434/v1/chat/completions",
		},
		{
			name:    "trailing slash",
			baseURL: "http://localhost:11434/",
			want:    "http://localhost:11434/v1/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildEndpointURL(tt.baseURL); got != tt.want {
				t.Fatalf("Expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestProviderBaseURLAccessors(t *testing.T) {
	provider := New("")
	if provider.DefaultBaseURL() != defaultBaseURL {
		t.Fatalf("Expected default base URL %q, got %q", defaultBaseURL, provider.DefaultBaseURL())
	}
	if provider.GetBaseURL() != defaultBaseURL {
		t.Fatalf("Expected initial base URL %q, got %q", defaultBaseURL, provider.GetBaseURL())
	}
	if provider.Provider.GetBaseURL() != buildEndpointURL(defaultBaseURL) {
		t.Fatalf("Expected embedded OpenAI endpoint %q, got %q", buildEndpointURL(defaultBaseURL), provider.Provider.GetBaseURL())
	}

	customURL := "http://127.0.0.1:11435"
	provider.SetBaseURL(customURL)
	if provider.GetBaseURL() != customURL {
		t.Fatalf("Expected custom base URL %q, got %q", customURL, provider.GetBaseURL())
	}
	if provider.Provider.GetBaseURL() != customURL {
		t.Fatalf("Expected custom embedded endpoint %q, got %q", customURL, provider.Provider.GetBaseURL())
	}
}
