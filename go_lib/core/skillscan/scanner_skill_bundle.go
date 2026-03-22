package skillscan

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go_lib/core"
	"go_lib/core/logging"
)

const (
	bundledScanSkillsEmbedRoot   = "bundled_scan_skills"
	bundledScanSkillsVersionFile = ".scan_bundle.version"
)

//go:embed bundled_scan_skills/**/*
var bundledScanSkillsFS embed.FS

// resolveDefaultScanSkillsRoot returns the default root directory for scan skills.
// Path strategy: {workspaceDir}/skills/skill_scanner
func resolveDefaultScanSkillsRoot() string {
	pm := core.GetPathManager()
	if pm != nil && pm.IsInitialized() {
		return pm.GetScanSkillDir()
	}
	return filepath.Join(os.TempDir(), "botsec", "skills", "skill_scanner")
}

// EnsureScanSkillsReleased ensures scanner skills are released to a disk directory
// and returns the release directory. Skills are released directly into targetRoot;
// if the version matches, the release is skipped.
func EnsureScanSkillsReleased(targetRoot string) (string, error) {
	if strings.TrimSpace(targetRoot) == "" {
		targetRoot = resolveDefaultScanSkillsRoot()
	}
	if err := os.MkdirAll(targetRoot, 0755); err != nil {
		return "", fmt.Errorf("create scan skills dir failed: %w", err)
	}

	desiredVersion, err := calculateScanSkillsVersion()
	if err != nil {
		return "", err
	}
	currentVersion, _ := os.ReadFile(filepath.Join(targetRoot, bundledScanSkillsVersionFile))
	if strings.TrimSpace(string(currentVersion)) == desiredVersion {
		logging.Info("[ScannerBundle] scan skills up-to-date, skip release: dir=%s, version=%s", targetRoot, desiredVersion[:12])
		return targetRoot, nil
	}

	// Version mismatch: clean skill subdirectories and re-release (preserve non-skill files)
	entries, _ := os.ReadDir(targetRoot)
	for _, entry := range entries {
		if entry.IsDir() {
			_ = os.RemoveAll(filepath.Join(targetRoot, entry.Name()))
		}
	}
	_ = os.Remove(filepath.Join(targetRoot, bundledScanSkillsVersionFile))

	if err := fs.WalkDir(bundledScanSkillsFS, bundledScanSkillsEmbedRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(bundledScanSkillsEmbedRoot, path)
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

		data, err := bundledScanSkillsFS.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, 0644)
	}); err != nil {
		return "", fmt.Errorf("release scan skills failed: %w", err)
	}

	if err := os.WriteFile(filepath.Join(targetRoot, bundledScanSkillsVersionFile), []byte(desiredVersion), 0644); err != nil {
		return "", fmt.Errorf("write scan bundle version failed: %w", err)
	}

	logging.Info("[ScannerBundle] scan skills released: dir=%s, version=%s", targetRoot, desiredVersion[:12])
	return targetRoot, nil
}

// calculateScanSkillsVersion calculates the SHA256 version hash for scanner skills.
func calculateScanSkillsVersion() (string, error) {
	var records []string
	err := fs.WalkDir(bundledScanSkillsFS, bundledScanSkillsEmbedRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		data, err := bundledScanSkillsFS.ReadFile(path)
		if err != nil {
			return err
		}
		hash := sha256.Sum256(data)
		records = append(records, path+"|"+hex.EncodeToString(hash[:]))
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk scan skills failed: %w", err)
	}

	sort.Strings(records)
	h := sha256.New()
	for _, r := range records {
		_, _ = h.Write([]byte(r))
		_, _ = h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// HasAnyScanSkill checks if a directory contains valid scan skills
func HasAnyScanSkill(root string) bool {
	entries, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, entry.Name(), "SKILL.md")); err == nil {
			return true
		}
	}
	return false
}
