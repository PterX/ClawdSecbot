//go:build windows

package sandbox

import (
	"path/filepath"
	"strings"
	"testing"
)

func envValue(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}

func TestBuildSandboxCommand_UsesPolicyDirAndLogDir(t *testing.T) {
	policyDir := t.TempDir()
	logDir := t.TempDir()

	mgr := NewSandboxManagerWithLogDir("Openclaw", policyDir, logDir)
	mgr.config = SandboxConfig{AssetName: "Openclaw"}
	mgr.gatewayArgs = []string{"gateway", "start"}
	mgr.gatewayEnv = []string{"TEST_ENV=1"}

	oldDLL := hookDLLPath
	hookDLLPath = `C:\test\sandbox_hook.dll`
	defer func() { hookDLLPath = oldDLL }()

	cmd, policyPath, err := mgr.buildSandboxCommand()
	if err != nil {
		t.Fatalf("buildSandboxCommand failed: %v", err)
	}

	if !strings.HasPrefix(policyPath, policyDir) {
		t.Fatalf("policy path should be under policyDir, got: %s", policyPath)
	}

	gotPolicyEnv := envValue(cmd.Env, "SANDBOX_POLICY_FILE")
	if gotPolicyEnv == "" {
		t.Fatalf("SANDBOX_POLICY_FILE not set")
	}
	if !strings.HasPrefix(gotPolicyEnv, policyDir) {
		t.Fatalf("SANDBOX_POLICY_FILE should be under policyDir, got: %s", gotPolicyEnv)
	}

	gotLogEnv := envValue(cmd.Env, "SANDBOX_LOG_FILE")
	if gotLogEnv == "" {
		t.Fatalf("SANDBOX_LOG_FILE not set")
	}
	if !strings.HasPrefix(gotLogEnv, logDir) {
		t.Fatalf("SANDBOX_LOG_FILE should be under logDir, got: %s", gotLogEnv)
	}
}

func TestBuildSandboxCommand_LogDirFallbackToPolicyDir(t *testing.T) {
	policyDir := t.TempDir()

	mgr := NewSandboxManager("Openclaw", policyDir)
	mgr.config = SandboxConfig{AssetName: "Openclaw"}
	mgr.gatewayArgs = []string{"gateway", "start"}

	oldDLL := hookDLLPath
	hookDLLPath = `C:\test\sandbox_hook.dll`
	defer func() { hookDLLPath = oldDLL }()

	cmd, _, err := mgr.buildSandboxCommand()
	if err != nil {
		t.Fatalf("buildSandboxCommand failed: %v", err)
	}

	gotLogEnv := envValue(cmd.Env, "SANDBOX_LOG_FILE")
	if gotLogEnv == "" {
		t.Fatalf("SANDBOX_LOG_FILE not set")
	}
	expectedPrefix := filepath.Clean(policyDir)
	if !strings.HasPrefix(filepath.Clean(gotLogEnv), expectedPrefix) {
		t.Fatalf("SANDBOX_LOG_FILE should fallback to policyDir, got: %s", gotLogEnv)
	}
}
