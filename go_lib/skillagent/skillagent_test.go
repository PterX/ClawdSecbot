package skillagent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_ParseMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	err := os.MkdirAll(skillDir, 0755)
	require.NoError(t, err)

	skillMd := `---
name: test-skill
description: A test skill for unit testing
version: 1.0.0
---

# Test Skill

This is a test skill.
`
	err = os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644)
	require.NoError(t, err)

	parser := NewParser()
	metadata, err := parser.ParseMetadata(filepath.Join(skillDir, "SKILL.md"))
	require.NoError(t, err)

	assert.Equal(t, "test-skill", metadata.Name)
	assert.Equal(t, "A test skill for unit testing", metadata.Description)
}

func TestParser_ParseManifest(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	err := os.MkdirAll(skillDir, 0755)
	require.NoError(t, err)

	skillMd := `---
name: full-skill
description: A skill with all fields
version: 2.0.0
allowed-tools:
  - read_skill_file
  - list_skill_files
model: gpt-4
tags:
  - testing
  - example
author: Test Author
---

# Full Skill

Instructions go here.
`
	err = os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644)
	require.NoError(t, err)

	parser := NewParser()
	manifest, err := parser.ParseManifest(filepath.Join(skillDir, "SKILL.md"))
	require.NoError(t, err)

	assert.Equal(t, "full-skill", manifest.Name)
	assert.Equal(t, "A skill with all fields", manifest.Description)
	assert.Equal(t, "2.0.0", manifest.Version)
	assert.Equal(t, []string{"read_skill_file", "list_skill_files"}, manifest.AllowedTools)
	assert.Equal(t, "gpt-4", manifest.Model)
	assert.Equal(t, []string{"testing", "example"}, manifest.Tags)
	assert.Equal(t, "Test Author", manifest.Author)
}

func TestParser_ParseContent(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	err := os.MkdirAll(skillDir, 0755)
	require.NoError(t, err)

	skillMd := `---
name: content-skill
description: A skill with content
---

# Content Skill

## Instructions

1. First step
2. Second step
3. Third step

## Notes

This is additional information.
`
	err = os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644)
	require.NoError(t, err)

	parser := NewParser()
	content, err := parser.ParseContent(filepath.Join(skillDir, "SKILL.md"))
	require.NoError(t, err)

	assert.Equal(t, "content-skill", content.Name)
	assert.Contains(t, content.Instructions, "# Content Skill")
	assert.Contains(t, content.Instructions, "First step")
	assert.Contains(t, content.Instructions, "Second step")
}

func TestParser_InvalidSkillMd(t *testing.T) {
	parser := NewParser()

	// Test missing file
	_, err := parser.ParseMetadata("/nonexistent/SKILL.md")
	assert.ErrorIs(t, err, ErrMissingSkillMd)

	// Test invalid frontmatter (missing ---)
	tmpDir := t.TempDir()
	invalidMd := filepath.Join(tmpDir, "SKILL.md")
	err = os.WriteFile(invalidMd, []byte("no frontmatter"), 0644)
	require.NoError(t, err)

	_, err = parser.ParseMetadata(invalidMd)
	assert.Error(t, err)
}

func TestParser_MissingRequiredFields(t *testing.T) {
	parser := NewParser()
	tmpDir := t.TempDir()

	// Test missing name
	noNameMd := filepath.Join(tmpDir, "no_name.md")
	err := os.WriteFile(noNameMd, []byte(`---
description: Has description but no name
---
`), 0644)
	require.NoError(t, err)

	_, err = parser.ParseMetadata(noNameMd)
	assert.Error(t, err)

	// Test missing description
	noDescMd := filepath.Join(tmpDir, "no_desc.md")
	err = os.WriteFile(noDescMd, []byte(`---
name: has-name
---
`), 0644)
	require.NoError(t, err)

	_, err = parser.ParseMetadata(noDescMd)
	assert.Error(t, err)
}

func TestParser_ScanDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, "skills")
	err := os.MkdirAll(skillsDir, 0755)
	require.NoError(t, err)

	// Create skill 1
	skill1Dir := filepath.Join(skillsDir, "skill-one")
	err = os.MkdirAll(skill1Dir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"), []byte(`---
name: skill-one
description: First test skill
---
`), 0644)
	require.NoError(t, err)

	// Create skill 2
	skill2Dir := filepath.Join(skillsDir, "skill-two")
	err = os.MkdirAll(skill2Dir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"), []byte(`---
name: skill-two
description: Second test skill
---
`), 0644)
	require.NoError(t, err)

	// Create a directory without SKILL.md (should be ignored)
	notSkillDir := filepath.Join(skillsDir, "not-a-skill")
	err = os.MkdirAll(notSkillDir, 0755)
	require.NoError(t, err)

	// Scan using Parser
	parser := NewParser()
	entries, err := os.ReadDir(skillsDir)
	require.NoError(t, err)

	var discovered []*SkillMetadata
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		mdPath := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
		md, parseErr := parser.ParseMetadata(mdPath)
		if parseErr != nil {
			continue
		}
		discovered = append(discovered, md)
	}

	assert.Len(t, discovered, 2)

	names := make(map[string]bool)
	for _, m := range discovered {
		names[m.Name] = true
	}
	assert.True(t, names["skill-one"])
	assert.True(t, names["skill-two"])
}

func TestSkill_IsToolAllowed(t *testing.T) {
	skill := NewSkill(&SkillMetadata{
		Name:        "test",
		Description: "test skill",
	})

	// No manifest - all tools allowed
	assert.True(t, skill.IsToolAllowed("any_tool"))

	// With manifest but no allowed tools - all tools allowed
	skill.SetManifest(&SkillManifest{
		SkillMetadata: *skill.Metadata,
	})
	assert.True(t, skill.IsToolAllowed("any_tool"))

	// With allowed tools - only those are allowed
	skill.SetManifest(&SkillManifest{
		SkillMetadata: *skill.Metadata,
		AllowedTools:  []string{"read_skill_file", "list_skill_files"},
	})
	assert.True(t, skill.IsToolAllowed("read_skill_file"))
	assert.True(t, skill.IsToolAllowed("list_skill_files"))
	assert.False(t, skill.IsToolAllowed("execute_script"))
}

func TestSkillContext(t *testing.T) {
	ctx := context.Background()

	// Create skill context
	skillCtx := NewSkillContext("test-skill", "/path/to/skill", "user input")

	// Test SetVariable and GetVariable
	skillCtx.SetVariable("key1", "value1")
	val, ok := skillCtx.GetVariable("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	// Test non-existent variable
	_, ok = skillCtx.GetVariable("nonexistent")
	assert.False(t, ok)

	// Test context value
	ctxWithSkill := WithSkillContext(ctx, skillCtx)
	retrieved, ok := GetSkillContext(ctxWithSkill)
	assert.True(t, ok)
	assert.Equal(t, "test-skill", retrieved.SkillName)

	// Test GetSkillPath and GetSkillName helpers
	assert.Equal(t, "/path/to/skill", GetSkillPath(ctxWithSkill))
	assert.Equal(t, "test-skill", GetSkillName(ctxWithSkill))

	// Test with context without skill context
	assert.Equal(t, "", GetSkillPath(ctx))
	assert.Equal(t, "", GetSkillName(ctx))
}

func TestExecuteOptions(t *testing.T) {
	opts := applyExecuteOptions([]ExecuteOption{
		WithForceSkill("my-skill"),
		WithAdditionalContext("extra context"),
		WithTimeout(30 * 1000000000), // 30s
		WithVariables(map[string]interface{}{"key": "value"}),
	})

	assert.Equal(t, "my-skill", opts.ForceSkill)
	assert.Equal(t, "extra context", opts.AdditionalContext)
	assert.Equal(t, 30*1000000000, int(opts.Timeout))
	assert.Equal(t, "value", opts.Variables["key"])
}

func TestStreamEventEmitter(t *testing.T) {
	ch := make(chan StreamEvent, 10)
	emitter := NewStreamEventEmitter(ch)

	// Test emitting events
	emitter.EmitSkillDiscovery([]*SkillMetadata{
		{Name: "skill1", Description: "desc1"},
	})
	emitter.EmitSkillSelected("skill1", "desc1")
	emitter.EmitToolCalling("read_file", `{"path": "test.txt"}`)
	emitter.EmitPartialOutput("partial")
	emitter.EmitFinalOutput("final")
	emitter.EmitComplete(true, 3)

	// Verify events
	event := <-ch
	assert.Equal(t, StreamEventSkillDiscovery, event.Type)

	event = <-ch
	assert.Equal(t, StreamEventSkillSelected, event.Type)

	event = <-ch
	assert.Equal(t, StreamEventToolCalling, event.Type)

	event = <-ch
	assert.Equal(t, StreamEventPartialOutput, event.Type)

	event = <-ch
	assert.Equal(t, StreamEventFinalOutput, event.Type)

	event = <-ch
	assert.Equal(t, StreamEventComplete, event.Type)
}

func TestSkillState(t *testing.T) {
	assert.Equal(t, "unloaded", SkillStateUnloaded.String())
	assert.Equal(t, "discovered", SkillStateDiscovered.String())
	assert.Equal(t, "activated", SkillStateActivated.String())
	assert.Equal(t, "executing", SkillStateExecuting.String())
	assert.Equal(t, "completed", SkillStateCompleted.String())
	assert.Equal(t, "failed", SkillStateFailed.String())
}
