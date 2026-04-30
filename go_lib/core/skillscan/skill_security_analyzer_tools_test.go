package skillscan

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListSkillFilesToolListsDirectoryEntries(t *testing.T) {
	root := t.TempDir()
	writeTestSkillFile(t, root, "SKILL.md", "name: demo\n")
	writeTestSkillFile(t, root, filepath.Join("nested", "script.py"), "print('hello')\n")

	tool := NewListSkillFilesTool(root)
	output, err := tool.InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("InvokableRun failed: %v", err)
	}

	var entries []struct {
		Path  string `json:"path"`
		IsDir bool   `json:"is_dir"`
	}
	if err := json.Unmarshal([]byte(output), &entries); err != nil {
		t.Fatalf("failed to decode entries: %v", err)
	}

	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.Path] = entry.IsDir
	}
	if _, ok := seen["SKILL.md"]; !ok {
		t.Fatalf("expected SKILL.md entry, got %#v", entries)
	}
	if isDir, ok := seen["nested"]; !ok || !isDir {
		t.Fatalf("expected nested directory entry, got %#v", entries)
	}
	if _, ok := seen[filepath.Join("nested", "script.py")]; !ok {
		t.Fatalf("expected nested file entry, got %#v", entries)
	}
}

func TestReadSkillFileToolRejectsTraversalAndAbsolutePaths(t *testing.T) {
	root := t.TempDir()
	writeTestSkillFile(t, root, "SKILL.md", "name: demo\n")
	tool := NewReadSkillFileTool(root)

	if _, err := tool.InvokableRun(context.Background(), `{"file_path":"../secret.txt"}`); err == nil {
		t.Fatal("expected path traversal to be rejected")
	}
	if _, err := tool.InvokableRun(context.Background(), `{"file_path":"/tmp/secret.txt"}`); err == nil {
		t.Fatal("expected absolute path to be rejected")
	}
}

func TestReadSkillFileToolReadsDirectoryAndSingleFileTargets(t *testing.T) {
	root := t.TempDir()
	writeTestSkillFile(t, root, "SKILL.md", "name: demo\n")
	dirTool := NewReadSkillFileTool(root)

	content, err := dirTool.InvokableRun(context.Background(), `{"file_path":"SKILL.md"}`)
	if err != nil {
		t.Fatalf("expected directory target read to succeed: %v", err)
	}
	if content != "name: demo\n" {
		t.Fatalf("unexpected directory target content: %q", content)
	}

	filePath := filepath.Join(t.TempDir(), "single.md")
	if err := os.WriteFile(filePath, []byte("single content"), 0600); err != nil {
		t.Fatalf("failed to write single file target: %v", err)
	}
	fileTool := NewReadSkillFileTool(filePath)
	content, err = fileTool.InvokableRun(context.Background(), `{"file_path":"single.md"}`)
	if err != nil {
		t.Fatalf("expected single file target read to succeed: %v", err)
	}
	if !strings.Contains(content, "single content") {
		t.Fatalf("unexpected single file content: %q", content)
	}
}

func TestParseAgentOutputParsesStructuredJSONBlock(t *testing.T) {
	output := "analysis\n```json\n{\"safe\":true,\"risk_level\":\"none\",\"summary\":\"no issues found\",\"issues\":[]}\n```"

	result, err := parseAgentOutput(output, "/tmp/demo")
	if err != nil {
		t.Fatalf("parseAgentOutput failed: %v", err)
	}
	if !result.Safe {
		t.Fatalf("expected structured JSON block to mark result safe: %#v", result)
	}
	if result.RiskLevel != "none" || result.Summary != "no issues found" {
		t.Fatalf("expected JSON block fields, got %#v", result)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("expected no structured issues, got %#v", result.Issues)
	}
	if result.RawOutput != output {
		t.Fatal("expected raw output to be preserved")
	}
}

func TestParseAgentOutputFallsBackToManualReviewForRiskText(t *testing.T) {
	result, err := parseAgentOutput("Summary:\nThis skill contains high risk command injection behavior.", "/skills/demo")
	if err != nil {
		t.Fatalf("parseAgentOutput failed: %v", err)
	}
	if result.Safe {
		t.Fatalf("risk indicator text should not default to safe: %#v", result)
	}
	if result.RiskLevel != "high" {
		t.Fatalf("expected high risk level, got %#v", result)
	}
	if len(result.Issues) != 1 || result.Issues[0].Type != "manual_review_required" {
		t.Fatalf("expected manual review issue, got %#v", result.Issues)
	}
}
