package skillscan

import (
	"fmt"
	"os"
	"path/filepath"
)

// SkillInfo represents a skill's information
type SkillInfo struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Hash       string `json:"hash"`
	Scanned    bool   `json:"scanned"`
	HasSkillMd bool   `json:"has_skill_md"`
}

// ListSkillsInDir lists all skills in a single directory.
// Each subdirectory is treated as a skill; SKILL.md presence and content hash
// are computed for every entry.
func ListSkillsInDir(skillsDir string) ([]SkillInfo, error) {
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read skills directory: %w", err)
	}

	var skills []SkillInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(skillsDir, entry.Name())
		skillMdPath := filepath.Join(skillPath, "SKILL.md")

		hasSkillMd := false
		if _, err := os.Stat(skillMdPath); err == nil {
			hasSkillMd = true
		}

		hash, err := CalculateSkillHash(skillPath)
		if err != nil {
			hash = ""
		}

		skills = append(skills, SkillInfo{
			Name:       entry.Name(),
			Path:       skillPath,
			Hash:       hash,
			Scanned:    false,
			HasSkillMd: hasSkillMd,
		})
	}

	return skills, nil
}
