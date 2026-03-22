package sandbox

import (
	"testing"
)

func TestValidateNetworkAddresses(t *testing.T) {
	tests := []struct {
		name      string
		addresses []string
		wantErr   int // expected number of errors
	}{
		{
			name:      "valid localhost addresses",
			addresses: []string{"localhost:8080", "localhost:443", "127.0.0.1:22"},
			wantErr:   0,
		},
		{
			name:      "valid wildcard address",
			addresses: []string{"*:80", "*:443"},
			wantErr:   0,
		},
		{
			name:      "invalid domain address",
			addresses: []string{"example.com:443"},
			wantErr:   1,
		},
		{
			name:      "invalid IP address",
			addresses: []string{"192.168.1.1:8080"},
			wantErr:   1,
		},
		{
			name:      "invalid CIDR notation",
			addresses: []string{"192.168.1.0/24:8080"},
			wantErr:   1,
		},
		{
			name:      "mixed valid and invalid",
			addresses: []string{"localhost:8080", "example.com:443", "192.168.1.1:80"},
			wantErr:   2,
		},
		{
			name:      "empty list",
			addresses: []string{},
			wantErr:   0,
		},
		{
			name:      "port only (invalid)",
			addresses: []string{":8080"},
			wantErr:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateNetworkAddresses(tt.addresses)
			if len(errs) != tt.wantErr {
				t.Errorf("ValidateNetworkAddresses() returned %d errors, want %d. Errors: %v", len(errs), tt.wantErr, errs)
			}
		})
	}
}

func TestIsValidSandboxNetworkAddress(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"*", true},
		{"*:80", true},
		{"*:443", true},
		{"localhost", true},
		{"localhost:8080", true},
		{"localhost:443", true},
		{"127.0.0.1", true},
		{"127.0.0.1:22", true},
		{"LOCALHOST:8080", true}, // case insensitive
		{"example.com", false},
		{"example.com:443", false},
		{"192.168.1.1", false},
		{"192.168.1.1:8080", false},
		{"10.0.0.1:443", false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			if got := isValidSandboxNetworkAddress(tt.addr); got != tt.want {
				t.Errorf("isValidSandboxNetworkAddress(%q) = %v, want %v", tt.addr, got, tt.want)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string // empty means skip (depends on home dir)
	}{
		{"/tmp", "/tmp"},
		{"/usr/bin", "/usr/bin"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := expandPath(tt.input)
			if tt.expected != "" && result != tt.expected {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatNetworkRule(t *testing.T) {
	tests := []struct {
		addr     string
		expected string
	}{
		{"localhost", "localhost:*"},
		{"localhost:8080", "localhost:8080"},
		{"*", "*:*"},
		{"*:443", "*:443"},
		{"example.com", "example.com:*"},
		{"example.com:8080", "example.com:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			result := formatNetworkRule(tt.addr)
			if result != tt.expected {
				t.Errorf("formatNetworkRule(%q) = %q, want %q", tt.addr, result, tt.expected)
			}
		})
	}
}

func TestSanitizeAssetName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test-asset", "test-asset"},
		{"my_asset", "my_asset"},
		{"My Asset", "My_Asset"},
		{"/path/to/asset", "_path_to_asset"},
		{"asset:8080", "asset_8080"},
		{"asset.name", "asset_name"},
		{"asset\\path", "asset_path"},
		{"  spaces  ", "__spaces__"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeAssetName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeAssetName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsIPAddress(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"127.0.0.1", true},
		{"8.8.8.8", true},
		{"localhost", false},
		{"example.com", false},
		{"*", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := isIPAddress(tt.host); got != tt.want {
				t.Errorf("isIPAddress(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}
