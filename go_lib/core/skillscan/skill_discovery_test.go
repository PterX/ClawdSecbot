package skillscan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListSkillsInDirReturnsNilForMissingDirectory(t *testing.T) {
	skills, err := ListSkillsInDir(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("expected missing directory to be ignored, got: %v", err)
	}
	if skills != nil {
		t.Fatalf("expected nil skills for missing directory, got %#v", skills)
	}
}

func TestListSkillsInDirListsOnlySkillDirectories(t *testing.T) {
	root := t.TempDir()
	writeTestSkillFile(t, root, filepath.Join("alpha", "SKILL.md"), "name: alpha\n")
	writeTestSkillFile(t, root, filepath.Join("alpha", "script.py"), "print('alpha')\n")
	writeTestSkillFile(t, root, filepath.Join("beta", "README.md"), "missing skill md\n")
	writeTestSkillFile(t, root, "not-a-skill.txt", "ignored")

	skills, err := ListSkillsInDir(root)
	if err != nil {
		t.Fatalf("ListSkillsInDir failed: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected two directory entries, got %#v", skills)
	}

	byName := map[string]SkillInfo{}
	for _, skill := range skills {
		byName[skill.Name] = skill
		if skill.Scanned {
			t.Fatalf("newly listed skill should not be marked scanned: %#v", skill)
		}
		if skill.Path == "" || !filepath.IsAbs(skill.Path) {
			t.Fatalf("expected absolute skill path, got %#v", skill)
		}
	}

	alpha := byName["alpha"]
	if !alpha.HasSkillMd {
		t.Fatalf("expected alpha to report SKILL.md presence: %#v", alpha)
	}
	if alpha.Hash == "" {
		t.Fatalf("expected alpha hash to be populated: %#v", alpha)
	}

	beta := byName["beta"]
	if beta.HasSkillMd {
		t.Fatalf("expected beta to report missing SKILL.md: %#v", beta)
	}
	if beta.Hash == "" {
		t.Fatalf("expected beta hash to be populated from existing files: %#v", beta)
	}
}

func TestHasAnyScanSkillRequiresSkillMarkdownInSubdirectory(t *testing.T) {
	root := t.TempDir()
	if HasAnyScanSkill(root) {
		t.Fatal("empty root should not contain scan skills")
	}

	writeTestSkillFile(t, root, "SKILL.md", "root file should not count")
	if HasAnyScanSkill(root) {
		t.Fatal("root-level SKILL.md should not count as a scan skill")
	}

	writeTestSkillFile(t, root, filepath.Join("scanner", "SKILL.md"), "name: scanner\n")
	if !HasAnyScanSkill(root) {
		t.Fatal("subdirectory with SKILL.md should count as a scan skill")
	}
}

func writeTestSkillFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()

	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create parent directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
}
