package skillagent

import (
	"path/filepath"
	"sync"
	"time"
)

// SkillState represents the current state of a skill in its lifecycle
type SkillState int

const (
	// SkillStateUnloaded indicates the skill has not been loaded yet
	SkillStateUnloaded SkillState = iota
	// SkillStateDiscovered indicates only metadata has been loaded
	SkillStateDiscovered
	// SkillStateActivated indicates the full manifest has been loaded
	SkillStateActivated
	// SkillStateExecuting indicates the skill is currently executing
	SkillStateExecuting
	// SkillStateCompleted indicates the skill execution has completed
	SkillStateCompleted
	// SkillStateFailed indicates the skill execution has failed
	SkillStateFailed
)

func (s SkillState) String() string {
	switch s {
	case SkillStateUnloaded:
		return "unloaded"
	case SkillStateDiscovered:
		return "discovered"
	case SkillStateActivated:
		return "activated"
	case SkillStateExecuting:
		return "executing"
	case SkillStateCompleted:
		return "completed"
	case SkillStateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// SkillMetadata represents the lightweight metadata loaded during discovery phase.
// This is the minimum information needed to identify and match a skill.
type SkillMetadata struct {
	// Name is the unique identifier of the skill (from frontmatter)
	Name string `yaml:"name" json:"name"`
	// Description describes when to use this skill (from frontmatter)
	Description string `yaml:"description" json:"description"`
	// Path is the filesystem path to the skill directory
	Path string `yaml:"-" json:"path"`
}

// SkillManifest extends SkillMetadata with additional configuration loaded during activation phase.
type SkillManifest struct {
	SkillMetadata `yaml:",inline"`
	// Version is the semantic version of the skill
	Version string `yaml:"version" json:"version,omitempty"`
	// AllowedTools is the list of tools this skill is allowed to use
	// If empty, all available tools are allowed
	AllowedTools []string `yaml:"allowed-tools" json:"allowed_tools,omitempty"`
	// Model is the recommended model for this skill
	Model string `yaml:"model" json:"model,omitempty"`
	// Tags are optional tags for categorizing the skill
	Tags []string `yaml:"tags" json:"tags,omitempty"`
	// Author is the skill author
	Author string `yaml:"author" json:"author,omitempty"`
}

// SkillContent extends SkillManifest with the full content loaded during execution phase.
type SkillContent struct {
	SkillManifest
	// Instructions is the full Markdown instructions from SKILL.md
	Instructions string `json:"instructions"`
	// Scripts is the list of script files in the scripts/ directory
	Scripts []string `json:"scripts,omitempty"`
	// Templates is the list of template files in the templates/ directory
	Templates []string `json:"templates,omitempty"`
	// References is the list of reference files in the references/ directory
	References []string `json:"references,omitempty"`
	// Assets is the list of asset files in the assets/ directory
	Assets []string `json:"assets,omitempty"`
}

// Skill represents a complete skill with all its state and content.
type Skill struct {
	mu sync.RWMutex

	// Metadata is always available after discovery
	Metadata *SkillMetadata
	// Manifest is available after activation
	Manifest *SkillManifest
	// Content is available after full loading
	Content *SkillContent

	// State is the current lifecycle state
	State SkillState
	// LastStateChange is the timestamp of the last state change
	LastStateChange time.Time
	// Error holds the last error if state is Failed
	Error error
}

// NewSkill creates a new Skill with the given metadata
func NewSkill(metadata *SkillMetadata) *Skill {
	return &Skill{
		Metadata:        metadata,
		State:           SkillStateDiscovered,
		LastStateChange: time.Now(),
	}
}

// GetState returns the current state of the skill
func (s *Skill) GetState() SkillState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}

// SetState updates the state of the skill
func (s *Skill) SetState(state SkillState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = state
	s.LastStateChange = time.Now()
}

// SetError sets the error and changes state to Failed
func (s *Skill) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Error = err
	s.State = SkillStateFailed
	s.LastStateChange = time.Now()
}

// GetName returns the skill name
func (s *Skill) GetName() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Metadata != nil {
		return s.Metadata.Name
	}
	return ""
}

// GetPath returns the skill path
func (s *Skill) GetPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Metadata != nil {
		return s.Metadata.Path
	}
	return ""
}

// GetDescription returns the skill description
func (s *Skill) GetDescription() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Metadata != nil {
		return s.Metadata.Description
	}
	return ""
}

// IsActivated returns true if the skill has been activated
func (s *Skill) IsActivated() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State >= SkillStateActivated && s.Manifest != nil
}

// IsFullyLoaded returns true if the skill content has been fully loaded
func (s *Skill) IsFullyLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Content != nil
}

// GetInstructions returns the skill instructions if loaded
func (s *Skill) GetInstructions() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Content != nil {
		return s.Content.Instructions
	}
	return ""
}

// GetAllowedTools returns the allowed tools list
func (s *Skill) GetAllowedTools() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Manifest != nil {
		return s.Manifest.AllowedTools
	}
	return nil
}

// IsToolAllowed checks if a tool is allowed for this skill
func (s *Skill) IsToolAllowed(toolName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If no manifest or no allowed tools specified, all tools are allowed
	if s.Manifest == nil || len(s.Manifest.AllowedTools) == 0 {
		return true
	}

	for _, allowed := range s.Manifest.AllowedTools {
		if allowed == toolName {
			return true
		}
	}
	return false
}

// SetManifest sets the manifest and updates state to Activated
func (s *Skill) SetManifest(manifest *SkillManifest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Manifest = manifest
	if s.State < SkillStateActivated {
		s.State = SkillStateActivated
		s.LastStateChange = time.Now()
	}
}

// SetContent sets the full content
func (s *Skill) SetContent(content *SkillContent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Content = content
}

// SkillMdPath returns the path to SKILL.md file
func (s *Skill) SkillMdPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Metadata != nil && s.Metadata.Path != "" {
		return filepath.Join(s.Metadata.Path, "SKILL.md")
	}
	return ""
}

// ScriptsDir returns the path to scripts directory
func (s *Skill) ScriptsDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Metadata != nil && s.Metadata.Path != "" {
		return filepath.Join(s.Metadata.Path, "scripts")
	}
	return ""
}

// TemplatesDir returns the path to templates directory
func (s *Skill) TemplatesDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Metadata != nil && s.Metadata.Path != "" {
		return filepath.Join(s.Metadata.Path, "templates")
	}
	return ""
}

// ReferencesDir returns the path to references directory
func (s *Skill) ReferencesDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Metadata != nil && s.Metadata.Path != "" {
		return filepath.Join(s.Metadata.Path, "references")
	}
	return ""
}

// AssetsDir returns the path to assets directory
func (s *Skill) AssetsDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Metadata != nil && s.Metadata.Path != "" {
		return filepath.Join(s.Metadata.Path, "assets")
	}
	return ""
}

// Clone creates a deep copy of the Skill (for safe concurrent access)
func (s *Skill) Clone() *Skill {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clone := &Skill{
		State:           s.State,
		LastStateChange: s.LastStateChange,
		Error:           s.Error,
	}

	if s.Metadata != nil {
		metaCopy := *s.Metadata
		clone.Metadata = &metaCopy
	}

	if s.Manifest != nil {
		manifestCopy := *s.Manifest
		if s.Manifest.AllowedTools != nil {
			manifestCopy.AllowedTools = make([]string, len(s.Manifest.AllowedTools))
			copy(manifestCopy.AllowedTools, s.Manifest.AllowedTools)
		}
		if s.Manifest.Tags != nil {
			manifestCopy.Tags = make([]string, len(s.Manifest.Tags))
			copy(manifestCopy.Tags, s.Manifest.Tags)
		}
		clone.Manifest = &manifestCopy
	}

	if s.Content != nil {
		contentCopy := *s.Content
		if s.Content.Scripts != nil {
			contentCopy.Scripts = make([]string, len(s.Content.Scripts))
			copy(contentCopy.Scripts, s.Content.Scripts)
		}
		if s.Content.Templates != nil {
			contentCopy.Templates = make([]string, len(s.Content.Templates))
			copy(contentCopy.Templates, s.Content.Templates)
		}
		if s.Content.References != nil {
			contentCopy.References = make([]string, len(s.Content.References))
			copy(contentCopy.References, s.Content.References)
		}
		if s.Content.Assets != nil {
			contentCopy.Assets = make([]string, len(s.Content.Assets))
			copy(contentCopy.Assets, s.Content.Assets)
		}
		clone.Content = &contentCopy
	}

	return clone
}
