package sandbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBlacklistModeGeneration tests blacklist mode policy generation
func TestBlacklistModeGeneration(t *testing.T) {
	homeDir, _ := os.UserHomeDir()
	downloadsPath := homeDir + "/Downloads"

	config := SandboxConfig{
		AssetName:         "TestAsset",
		GatewayBinaryPath: "/usr/local/bin/openclaw",
		GatewayConfigPath: homeDir + "/.config/openclaw.json",
		PathPermission: PathPermissionConfig{
			Mode:  ModeBlacklist,
			Paths: []string{downloadsPath},
		},
		NetworkPermission: NetworkPermissionConfig{
			Outbound: DirectionalNetworkConfig{
				Mode:      ModeBlacklist,
				Addresses: []string{},
			},
		},
		ShellPermission: ShellPermissionConfig{
			Mode:     ModeBlacklist,
			Commands: []string{},
		},
	}

	policy := NewSeatbeltPolicy(config)
	content, err := policy.GeneratePolicy()
	if err != nil {
		t.Fatalf("Failed to generate policy: %v", err)
	}

	// Verify key elements are present and in correct order
	if !strings.Contains(content, "(version 1)") {
		t.Error("Missing version declaration")
	}

	if !strings.Contains(content, "(debug deny)") {
		t.Error("Missing debug comment")
	}

	// Check that deny comes before allow default
	denyIndex := strings.Index(content, "(deny file-read-data file-read-metadata file-write*")
	allowIndex := strings.Index(content, "(allow default)")

	if denyIndex == -1 {
		t.Error("Missing deny rule for Downloads")
	}

	if allowIndex == -1 {
		t.Error("Missing allow default")
	}

	if denyIndex > allowIndex {
		t.Error("CRITICAL: deny rule must come BEFORE (allow default) in blacklist mode")
	}

	// Verify absolute path rule is generated and no "~/" remains.
	if !strings.Contains(content, `(subpath "`) {
		t.Error("Should contain subpath rule")
	}
	if strings.Contains(content, "(subpath \"~/") {
		t.Error("Path should be expanded, not use ~")
	}

	// Verify both read-data and read-metadata are denied
	if !strings.Contains(content, "file-read-data") {
		t.Error("Missing file-read-data in deny rule")
	}
	if !strings.Contains(content, "file-read-metadata") {
		t.Error("Missing file-read-metadata in deny rule")
	}

	t.Logf("Generated policy:\n%s", content)
}

// TestWhitelistModeGeneration tests whitelist mode policy generation
func TestWhitelistModeGeneration(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	config := SandboxConfig{
		AssetName:         "TestAsset",
		GatewayBinaryPath: "/usr/local/bin/openclaw",
		GatewayConfigPath: homeDir + "/.config/openclaw.json",
		PathPermission: PathPermissionConfig{
			Mode:  ModeWhitelist,
			Paths: []string{homeDir + "/allowed"},
		},
		NetworkPermission: NetworkPermissionConfig{
			Outbound: DirectionalNetworkConfig{
				Mode:      ModeWhitelist,
				Addresses: []string{"api.example.com:443"},
			},
		},
		ShellPermission: ShellPermissionConfig{
			Mode:     ModeWhitelist,
			Commands: []string{"/usr/bin/curl"},
		},
	}

	policy := NewSeatbeltPolicy(config)
	content, err := policy.GeneratePolicy()
	if err != nil {
		t.Fatalf("Failed to generate policy: %v", err)
	}

	// Verify whitelist mode structure
	if !strings.Contains(content, "(deny default)") {
		t.Error("Whitelist mode must start with (deny default)")
	}

	// Check deny default comes before allow rules
	denyDefaultIndex := strings.Index(content, "(deny default)")
	allowIndex := strings.Index(content, "(allow ")

	if denyDefaultIndex == -1 || allowIndex == -1 {
		t.Error("Missing deny default or allow rules")
	}

	if denyDefaultIndex > allowIndex {
		t.Error("(deny default) must come before (allow ...) in whitelist mode")
	}

	t.Logf("Generated policy:\n%s", content)
}

// TestPathExpansion tests path expansion to absolute paths
func TestPathExpansion(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    homeDir + "/Downloads",
			expected: homeDir + "/Downloads", // Now returns absolute path
		},
		{
			input:    "~/Documents",
			expected: filepath.Join(homeDir, "Documents"), // Expands ~ to absolute
		},
		{
			input:    "/tmp",
			expected: "/tmp",
		},
		{
			input:    homeDir,
			expected: homeDir,
		},
	}

	for _, tt := range tests {
		result := expandPath(tt.input)
		if result != tt.expected {
			t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
