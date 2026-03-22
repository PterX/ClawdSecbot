package shepherd

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go_lib/core"
	"go_lib/core/logging"
	"go_lib/skillagent"
)

const (
	bundledSkillsEmbedRoot   = "bundled_react_skills"
	bundledSkillsVersionFile = ".bundle.version"
)

//go:embed bundled_react_skills/**/*
var bundledReActSkillsFS embed.FS

// resolveDefaultReActSkillsRoot returns the default root directory for ReAct guard skills.
func resolveDefaultReActSkillsRoot() string {
	pm := core.GetPathManager()
	if pm != nil && pm.IsInitialized() {
		return pm.GetReActSkillDir()
	}
	return filepath.Join(os.TempDir(), "botsec", "skills", "shepherd_gate")
}

// ensureBundledReActSkillsReleased ensures bundled skills are released to disk.
func ensureBundledReActSkillsReleased(targetRoot string) (string, error) {
	if strings.TrimSpace(targetRoot) == "" {
		targetRoot = resolveDefaultReActSkillsRoot()
	}
	if err := os.MkdirAll(targetRoot, 0755); err != nil {
		return "", fmt.Errorf("create skills dir failed: %w", err)
	}

	desiredVersion, err := calculateBundledSkillsVersion()
	if err != nil {
		return "", err
	}
	currentVersion, _ := os.ReadFile(filepath.Join(targetRoot, bundledSkillsVersionFile))
	if strings.TrimSpace(string(currentVersion)) == desiredVersion {
		logging.ShepherdGateInfo("[ShepherdGate][SkillBundle] skills up-to-date, skip release: dir=%s, version=%s", targetRoot, desiredVersion[:12])
		return targetRoot, nil
	}

	entries, _ := os.ReadDir(targetRoot)
	for _, entry := range entries {
		if entry.IsDir() {
			_ = os.RemoveAll(filepath.Join(targetRoot, entry.Name()))
		}
	}
	_ = os.Remove(filepath.Join(targetRoot, bundledSkillsVersionFile))

	if err := fs.WalkDir(bundledReActSkillsFS, bundledSkillsEmbedRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(bundledSkillsEmbedRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		targetPath := filepath.Join(targetRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		data, err := bundledReActSkillsFS.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, 0644)
	}); err != nil {
		return "", fmt.Errorf("release bundled skills failed: %w", err)
	}

	if err := os.WriteFile(filepath.Join(targetRoot, bundledSkillsVersionFile), []byte(desiredVersion), 0644); err != nil {
		return "", fmt.Errorf("write bundle version failed: %w", err)
	}

	logging.ShepherdGateInfo("[ShepherdGate][SkillBundle] bundled skills released: dir=%s, version=%s", targetRoot, desiredVersion[:12])
	return targetRoot, nil
}

// ListBundledReActSkillsInternal lists all bundled ReAct guard skill metadata.
func ListBundledReActSkillsInternal() string {
	dir, err := ensureBundledReActSkillsReleased("")
	if err != nil {
		result, _ := json.Marshal(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("release bundled skills failed: %v", err),
		})
		return string(result)
	}

	parser := skillagent.NewParser()
	entries, err := os.ReadDir(dir)
	if err != nil {
		result, _ := json.Marshal(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("read skills dir failed: %v", err),
		})
		return string(result)
	}

	type skillItem struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	var items []skillItem
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		mdPath := filepath.Join(dir, entry.Name(), "SKILL.md")
		md, err := parser.ParseMetadata(mdPath)
		if err != nil {
			continue
		}
		items = append(items, skillItem{Name: md.Name, Description: md.Description})
	}

	result, _ := json.Marshal(map[string]interface{}{
		"success": true,
		"data":    items,
	})
	return string(result)
}

func calculateBundledSkillsVersion() (string, error) {
	var records []string
	err := fs.WalkDir(bundledReActSkillsFS, bundledSkillsEmbedRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		data, err := bundledReActSkillsFS.ReadFile(path)
		if err != nil {
			return err
		}
		hash := sha256.Sum256(data)
		records = append(records, path+"|"+hex.EncodeToString(hash[:]))
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk bundled skills failed: %w", err)
	}

	sort.Strings(records)
	h := sha256.New()
	for _, r := range records {
		_, _ = h.Write([]byte(r))
		_, _ = h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
