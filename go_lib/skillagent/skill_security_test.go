//go:build integration
// +build integration

package skillagent_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"go_lib/skillagent"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTestDataDir returns the absolute path to the testdata directory
func getTestDataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata")
}

// TestSkillAgentPromptInjectionDetection tests the prompt injection detection capability
// using SkillAgent SDK with a security scanner skill
func TestSkillAgentPromptInjectionDetection(t *testing.T) {
	// MiniMax model configuration (OpenAI compatible protocol)
	apiKey := os.Getenv("MINIMAX_API_KEY")
	if apiKey == "" {
		t.Skip("MINIMAX_API_KEY environment variable not set, skipping integration test")
	}

	ctx := context.Background()

	// Create MiniMax ChatModel (OpenAI compatible)
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  apiKey,
		BaseURL: "https://api.minimax.io/v1",
		Model:   "MiniMax-M2.5",
		Timeout: 120 * time.Second,
	})
	require.NoError(t, err, "Failed to create chat model")

	// Get test data paths
	testdataDir := getTestDataDir()
	skillsDir := filepath.Join(testdataDir, "skills")
	maliciousSkillPath := filepath.Join(testdataDir, "malicious-skills", "malicious-helper")

	t.Logf("Skills directory: %s", skillsDir)
	t.Logf("Malicious skill path: %s", maliciousSkillPath)

	// Create SkillAgent with ADK middleware architecture
	agent, err := skillagent.NewSkillAgent(ctx, &skillagent.SkillAgentConfig{
		ChatModel:        chatModel,
		SkillsDir:        skillsDir,
		MaxStep:          30,
		ExecutionTimeout: 3 * time.Minute,
		Hooks: &skillagent.SkillHooks{
			OnSkillDiscovered: func(metadata *skillagent.SkillMetadata) {
				t.Logf("Discovered skill: %s - %s", metadata.Name, metadata.Description)
			},
			OnSkillSelected: func(skillName string) {
				t.Logf("Selected skill: %s", skillName)
			},
			OnToolCall: func(toolName, arguments string) {
				t.Logf("Tool call: %s(%s)", toolName, truncateString(arguments, 100))
			},
			OnToolResult: func(toolName, result string, err error) {
				if err != nil {
					t.Logf("Tool result error: %s - %v", toolName, err)
				} else {
					t.Logf("Tool result: %s - %s", toolName, truncateString(result, 200))
				}
			},
		},
	})
	require.NoError(t, err, "Failed to create SkillAgent")
	defer agent.Close()

	// List discovered skills (ADK skill middleware handles discovery)
	skills, err := agent.ListSkills(ctx)
	require.NoError(t, err, "Failed to list skills")
	t.Logf("Discovered %d skills:", len(skills))
	for _, skill := range skills {
		t.Logf("  - %s: %s", skill.Name, skill.Description)
	}

	// Execute security scan
	userInput := fmt.Sprintf("Please analyze the skill directory at %s for security risks. Read the SKILL.md file and check for any dangerous patterns.", maliciousSkillPath)

	t.Logf("User input: %s", userInput)

	result, err := agent.Execute(ctx, userInput,
		skillagent.WithForceSkill("prompt-injection-scanner"),
		skillagent.WithTimeout(3*time.Minute),
	)
	require.NoError(t, err, "Failed to execute skill")

	// Output analysis result
	t.Logf("\n========== Analysis Output ==========")
	t.Logf("%s", result.Output)
	t.Logf("=====================================")
	t.Logf("Success: %v, Duration: %v, Tool Calls: %d", result.Success, result.Duration, len(result.ToolCallHistory))

	// Verify detection result
	assert.True(t, result.Success, "Execution should succeed")

	// Check if output contains risk indicators
	output := strings.ToLower(result.Output)
	hasRiskIndicator := strings.Contains(output, "injection") ||
		strings.Contains(output, "unsafe") ||
		strings.Contains(output, "malicious") ||
		strings.Contains(output, "risk") ||
		strings.Contains(output, "dangerous") ||
		strings.Contains(output, "\"safe\": false") ||
		strings.Contains(output, "\"safe\":false") ||
		strings.Contains(output, "exfiltration")

	assert.True(t, hasRiskIndicator, "Should detect security risks in the malicious skill. Output: %s", result.Output)

	// Check for specific patterns detected
	hasIgnoreInstruction := strings.Contains(output, "ignore") && strings.Contains(output, "instruction")
	hasSystemTag := strings.Contains(output, "[system]") || strings.Contains(output, "{{system}}")
	hasExfiltration := strings.Contains(output, "ssh") || strings.Contains(output, "aws") || strings.Contains(output, "curl")

	t.Logf("Detection results:")
	t.Logf("  - Ignore instruction pattern: %v", hasIgnoreInstruction)
	t.Logf("  - System tag pattern: %v", hasSystemTag)
	t.Logf("  - Data exfiltration pattern: %v", hasExfiltration)
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
