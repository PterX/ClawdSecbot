package skillagent

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// frontmatterDelimiter is the YAML frontmatter delimiter
	frontmatterDelimiter = "---"
	// maxMetadataSize is the maximum size to read for metadata extraction
	maxMetadataSize = 8192 // 8KB should be enough for frontmatter
)

// Parser handles parsing of SKILL.md files with progressive disclosure support
type Parser struct{}

// NewParser creates a new Parser instance
func NewParser() *Parser {
	return &Parser{}
}

// ParseMetadata parses only the name and description from SKILL.md (discovery phase).
// This is optimized for minimal I/O - reads only until frontmatter ends.
func (p *Parser) ParseMetadata(skillMdPath string) (*SkillMetadata, error) {
	file, err := os.Open(skillMdPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrMissingSkillMd
		}
		return nil, fmt.Errorf("failed to open SKILL.md: %w", err)
	}
	defer file.Close()

	frontmatter, _, err := p.extractFrontmatter(file)
	if err != nil {
		return nil, err
	}

	var metadata SkillMetadata
	if err := yaml.Unmarshal(frontmatter, &metadata); err != nil {
		return nil, NewParseError(skillMdPath, 0, "invalid YAML frontmatter", err)
	}

	if metadata.Name == "" {
		return nil, NewParseError(skillMdPath, 0, "missing required field: name", ErrMissingRequiredField)
	}
	if metadata.Description == "" {
		return nil, NewParseError(skillMdPath, 0, "missing required field: description", ErrMissingRequiredField)
	}

	return &metadata, nil
}

// ParseManifest parses the full YAML frontmatter from SKILL.md (activation phase).
func (p *Parser) ParseManifest(skillMdPath string) (*SkillManifest, error) {
	file, err := os.Open(skillMdPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrMissingSkillMd
		}
		return nil, fmt.Errorf("failed to open SKILL.md: %w", err)
	}
	defer file.Close()

	frontmatter, _, err := p.extractFrontmatter(file)
	if err != nil {
		return nil, err
	}

	var manifest SkillManifest
	if err := yaml.Unmarshal(frontmatter, &manifest); err != nil {
		return nil, NewParseError(skillMdPath, 0, "invalid YAML frontmatter", err)
	}

	if manifest.Name == "" {
		return nil, NewParseError(skillMdPath, 0, "missing required field: name", ErrMissingRequiredField)
	}
	if manifest.Description == "" {
		return nil, NewParseError(skillMdPath, 0, "missing required field: description", ErrMissingRequiredField)
	}

	return &manifest, nil
}

// ParseContent parses the complete SKILL.md including Markdown instructions (execution phase).
func (p *Parser) ParseContent(skillMdPath string) (*SkillContent, error) {
	data, err := os.ReadFile(skillMdPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrMissingSkillMd
		}
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	frontmatter, markdown, err := p.splitFrontmatterAndContent(data)
	if err != nil {
		return nil, err
	}

	var content SkillContent
	if err := yaml.Unmarshal(frontmatter, &content.SkillManifest); err != nil {
		return nil, NewParseError(skillMdPath, 0, "invalid YAML frontmatter", err)
	}

	if content.Name == "" {
		return nil, NewParseError(skillMdPath, 0, "missing required field: name", ErrMissingRequiredField)
	}
	if content.Description == "" {
		return nil, NewParseError(skillMdPath, 0, "missing required field: description", ErrMissingRequiredField)
	}

	content.Instructions = strings.TrimSpace(string(markdown))

	return &content, nil
}

// extractFrontmatter reads the YAML frontmatter from a reader.
// It returns the frontmatter bytes and the line number where content starts.
func (p *Parser) extractFrontmatter(r io.Reader) ([]byte, int, error) {
	scanner := bufio.NewScanner(r)
	var frontmatterLines []string
	inFrontmatter := false
	lineNum := 0
	contentStartLine := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if lineNum == 1 {
			// First line must be ---
			if strings.TrimSpace(line) != frontmatterDelimiter {
				return nil, 0, NewParseError("", 1, "SKILL.md must start with ---", ErrInvalidFrontmatter)
			}
			inFrontmatter = true
			continue
		}

		if inFrontmatter {
			if strings.TrimSpace(line) == frontmatterDelimiter {
				// End of frontmatter
				contentStartLine = lineNum + 1
				break
			}
			frontmatterLines = append(frontmatterLines, line)
		}

		// Safety limit - frontmatter shouldn't be huge
		if lineNum > 500 {
			return nil, 0, NewParseError("", lineNum, "frontmatter too long (>500 lines)", ErrInvalidFrontmatter)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, fmt.Errorf("error reading file: %w", err)
	}

	if len(frontmatterLines) == 0 {
		return nil, 0, NewParseError("", 0, "empty frontmatter", ErrInvalidFrontmatter)
	}

	frontmatter := []byte(strings.Join(frontmatterLines, "\n"))
	return frontmatter, contentStartLine, nil
}

// splitFrontmatterAndContent splits the file content into frontmatter and markdown parts.
func (p *Parser) splitFrontmatterAndContent(data []byte) (frontmatter, markdown []byte, err error) {
	// Find the frontmatter delimiters
	lines := bytes.Split(data, []byte("\n"))
	if len(lines) == 0 {
		return nil, nil, NewParseError("", 0, "empty file", ErrInvalidSkillFormat)
	}

	// First line must be ---
	if !bytes.Equal(bytes.TrimSpace(lines[0]), []byte(frontmatterDelimiter)) {
		return nil, nil, NewParseError("", 1, "SKILL.md must start with ---", ErrInvalidFrontmatter)
	}

	// Find the closing ---
	endIndex := -1
	for i := 1; i < len(lines); i++ {
		if bytes.Equal(bytes.TrimSpace(lines[i]), []byte(frontmatterDelimiter)) {
			endIndex = i
			break
		}
	}

	if endIndex == -1 {
		return nil, nil, NewParseError("", 0, "missing closing --- for frontmatter", ErrInvalidFrontmatter)
	}

	// Extract frontmatter (between the two ---)
	frontmatterLines := lines[1:endIndex]
	frontmatter = bytes.Join(frontmatterLines, []byte("\n"))

	// Extract markdown content (after the closing ---)
	if endIndex+1 < len(lines) {
		markdownLines := lines[endIndex+1:]
		markdown = bytes.Join(markdownLines, []byte("\n"))
	}

	return frontmatter, markdown, nil
}

// ValidateMetadata validates that required metadata fields are present
func (p *Parser) ValidateMetadata(metadata *SkillMetadata) error {
	if metadata == nil {
		return NewValidationError("metadata", "metadata is nil")
	}
	if metadata.Name == "" {
		return NewValidationError("name", "name is required")
	}
	if metadata.Description == "" {
		return NewValidationError("description", "description is required")
	}
	return nil
}

// ValidateManifest validates the manifest fields
func (p *Parser) ValidateManifest(manifest *SkillManifest) error {
	if manifest == nil {
		return NewValidationError("manifest", "manifest is nil")
	}
	if err := p.ValidateMetadata(&manifest.SkillMetadata); err != nil {
		return err
	}
	return nil
}

// ParseMetadataFromBytes parses metadata from raw bytes (useful for testing)
func (p *Parser) ParseMetadataFromBytes(data []byte) (*SkillMetadata, error) {
	frontmatter, _, err := p.splitFrontmatterAndContent(data)
	if err != nil {
		return nil, err
	}

	var metadata SkillMetadata
	if err := yaml.Unmarshal(frontmatter, &metadata); err != nil {
		return nil, NewParseError("", 0, "invalid YAML frontmatter", err)
	}

	if err := p.ValidateMetadata(&metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// ParseManifestFromBytes parses manifest from raw bytes (useful for testing)
func (p *Parser) ParseManifestFromBytes(data []byte) (*SkillManifest, error) {
	frontmatter, _, err := p.splitFrontmatterAndContent(data)
	if err != nil {
		return nil, err
	}

	var manifest SkillManifest
	if err := yaml.Unmarshal(frontmatter, &manifest); err != nil {
		return nil, NewParseError("", 0, "invalid YAML frontmatter", err)
	}

	if err := p.ValidateManifest(&manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// ParseContentFromBytes parses full content from raw bytes (useful for testing)
func (p *Parser) ParseContentFromBytes(data []byte) (*SkillContent, error) {
	frontmatter, markdown, err := p.splitFrontmatterAndContent(data)
	if err != nil {
		return nil, err
	}

	var content SkillContent
	if err := yaml.Unmarshal(frontmatter, &content.SkillManifest); err != nil {
		return nil, NewParseError("", 0, "invalid YAML frontmatter", err)
	}

	if err := p.ValidateManifest(&content.SkillManifest); err != nil {
		return nil, err
	}

	content.Instructions = strings.TrimSpace(string(markdown))

	return &content, nil
}
