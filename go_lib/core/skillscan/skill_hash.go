package skillscan

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SkillScanResult represents the result of scanning a skill for prompt injection
type SkillScanResult struct {
	SkillName string   `json:"skill_name"`
	SkillHash string   `json:"skill_hash"`
	Safe      bool     `json:"safe"`
	Issues    []string `json:"issues,omitempty"`
}

// skipTransientFile returns true for OS/toolchain-generated files that should
// not participate in hash computation (they may appear or vanish between runs).
func skipTransientFile(name string) bool {
	switch name {
	case ".DS_Store", "Thumbs.db", "desktop.ini", ".gitkeep":
		return true
	}
	return false
}

// skipTransientDir returns true for directories whose contents are
// auto-generated and should not affect the skill hash.
func skipTransientDir(name string) bool {
	switch name {
	case ".git", "__pycache__", "node_modules", ".venv", ".mypy_cache",
		".pytest_cache", ".ruff_cache", ".tox":
		return true
	}
	return false
}

// CalculateSkillHash calculates the hash of a skill by sorting all files
// alphabetically and computing a combined SHA-256 hash.
// Transient/OS-generated files and directories are excluded for stability.
func CalculateSkillHash(skillPath string) (string, error) {
	var files []string

	// Walk the directory and collect all files
	err := filepath.Walk(skillPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := info.Name()
		if info.IsDir() {
			if skipTransientDir(name) {
				return filepath.SkipDir
			}
			return nil
		}
		if skipTransientFile(name) {
			return nil
		}
		// Store relative path for consistent sorting
		relPath, err := filepath.Rel(skillPath, path)
		if err != nil {
			return err
		}
		files = append(files, relPath)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to walk skill directory: %w", err)
	}

	// Sort files alphabetically for consistent hash
	sort.Strings(files)

	// Calculate combined hash
	hasher := sha256.New()
	for _, relPath := range files {
		fullPath := filepath.Join(skillPath, relPath)

		// Add file path to hash for structure integrity
		hasher.Write([]byte(relPath))

		// Add file content to hash
		f, err := os.Open(fullPath)
		if err != nil {
			return "", fmt.Errorf("failed to open file %s: %w", relPath, err)
		}

		if _, err := io.Copy(hasher, f); err != nil {
			f.Close()
			return "", fmt.Errorf("failed to read file %s: %w", relPath, err)
		}
		f.Close()
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// ScanSkillForPromptInjection scans a skill for prompt injection vulnerabilities
// using pattern-based detection.
func ScanSkillForPromptInjection(skillPath string) (*SkillScanResult, error) {
	skillName := filepath.Base(skillPath)

	hash, err := CalculateSkillHash(skillPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate skill hash: %w", err)
	}

	result := &SkillScanResult{
		SkillName: skillName,
		SkillHash: hash,
		Safe:      true,
		Issues:    []string{},
	}

	issues := DetectPromptInjectionPatterns(skillPath)
	if len(issues) > 0 {
		result.Safe = false
		result.Issues = issues
	}

	return result, nil
}

// DetectPromptInjectionPatterns scans skill files for common prompt injection patterns
func DetectPromptInjectionPatterns(skillPath string) []string {
	var issues []string

	// Common prompt injection patterns to detect
	dangerousPatterns := []struct {
		pattern     string
		description string
	}{
		{"ignore previous instructions", "Attempt to override system instructions"},
		{"ignore all previous", "Attempt to override system instructions"},
		{"disregard your instructions", "Attempt to override system instructions"},
		{"forget your instructions", "Attempt to override system instructions"},
		{"you are now", "Role hijacking attempt"},
		{"act as if you", "Role hijacking attempt"},
		{"pretend you are", "Role hijacking attempt"},
		{"new instructions:", "Instruction injection attempt"},
		{"system prompt:", "System prompt manipulation"},
		{"[system]", "Hidden system directive"},
		{"<system>", "Hidden system directive"},
		{"{{system}}", "Template injection attempt"},
		{"%system%", "Template injection attempt"},
	}

	// Walk through all files in the skill
	filepath.Walk(skillPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Only scan text files
		ext := strings.ToLower(filepath.Ext(path))
		textExts := map[string]bool{
			".md": true, ".txt": true, ".yaml": true, ".yml": true,
			".json": true, ".py": true, ".js": true, ".ts": true,
		}
		if !textExts[ext] {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		contentLower := strings.ToLower(string(content))
		relPath, _ := filepath.Rel(skillPath, path)

		for _, dp := range dangerousPatterns {
			if strings.Contains(contentLower, dp.pattern) {
				issues = append(issues, fmt.Sprintf("[%s] %s: detected pattern '%s'", relPath, dp.description, dp.pattern))
			}
		}

		return nil
	})

	return issues
}
