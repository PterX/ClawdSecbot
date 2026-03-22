package skillagent

import (
	"context"
	"sync"
	"time"
)

// contextKey is the type for context keys used by this package
type contextKey string

const (
	// skillContextKey is the context key for SkillContext
	skillContextKey contextKey = "skill_context"
)

// SkillContext holds the execution context for a skill invocation.
// It is passed through context.Context and accessible by tools.
type SkillContext struct {
	mu sync.RWMutex

	// SkillName is the name of the currently executing skill
	SkillName string

	// SkillPath is the filesystem path to the skill directory
	SkillPath string

	// UserInput is the original user input that triggered this execution
	UserInput string

	// Variables holds variables that can be used in templates
	Variables map[string]interface{}

	// ToolResults holds results from previous tool calls (keyed by tool name + call index)
	ToolResults map[string]string

	// Metadata holds custom metadata set by tools or hooks
	Metadata map[string]interface{}

	// StartTime is when the execution started
	StartTime time.Time

	// ToolCallCount tracks the number of tool calls made
	ToolCallCount int
}

// NewSkillContext creates a new SkillContext
func NewSkillContext(skillName, skillPath, userInput string) *SkillContext {
	return &SkillContext{
		SkillName:   skillName,
		SkillPath:   skillPath,
		UserInput:   userInput,
		Variables:   make(map[string]interface{}),
		ToolResults: make(map[string]string),
		Metadata:    make(map[string]interface{}),
		StartTime:   time.Now(),
	}
}

// SetVariable sets a variable in the context
func (sc *SkillContext) SetVariable(key string, value interface{}) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.Variables[key] = value
}

// GetVariable gets a variable from the context
func (sc *SkillContext) GetVariable(key string) (interface{}, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	v, ok := sc.Variables[key]
	return v, ok
}

// SetToolResult stores a tool result
func (sc *SkillContext) SetToolResult(toolName string, result string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	key := toolName + "_" + string(rune(sc.ToolCallCount))
	sc.ToolResults[key] = result
	sc.ToolCallCount++
}

// GetToolResult gets the most recent result for a tool
func (sc *SkillContext) GetToolResult(toolName string) (string, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	// Find the latest result for this tool
	for i := sc.ToolCallCount - 1; i >= 0; i-- {
		key := toolName + "_" + string(rune(i))
		if result, ok := sc.ToolResults[key]; ok {
			return result, true
		}
	}
	return "", false
}

// SetMetadata sets a metadata value
func (sc *SkillContext) SetMetadata(key string, value interface{}) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.Metadata[key] = value
}

// GetMetadata gets a metadata value
func (sc *SkillContext) GetMetadata(key string) (interface{}, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	v, ok := sc.Metadata[key]
	return v, ok
}

// ElapsedTime returns the time elapsed since execution started
func (sc *SkillContext) ElapsedTime() time.Duration {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return time.Since(sc.StartTime)
}

// GetToolCallCount returns the number of tool calls made
func (sc *SkillContext) GetToolCallCount() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.ToolCallCount
}

// Clone creates a shallow copy of the SkillContext
func (sc *SkillContext) Clone() *SkillContext {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	clone := &SkillContext{
		SkillName:     sc.SkillName,
		SkillPath:     sc.SkillPath,
		UserInput:     sc.UserInput,
		StartTime:     sc.StartTime,
		ToolCallCount: sc.ToolCallCount,
		Variables:     make(map[string]interface{}),
		ToolResults:   make(map[string]string),
		Metadata:      make(map[string]interface{}),
	}

	for k, v := range sc.Variables {
		clone.Variables[k] = v
	}
	for k, v := range sc.ToolResults {
		clone.ToolResults[k] = v
	}
	for k, v := range sc.Metadata {
		clone.Metadata[k] = v
	}

	return clone
}

// WithSkillContext attaches a SkillContext to a context.Context
func WithSkillContext(ctx context.Context, sc *SkillContext) context.Context {
	return context.WithValue(ctx, skillContextKey, sc)
}

// GetSkillContext retrieves the SkillContext from a context.Context
func GetSkillContext(ctx context.Context) (*SkillContext, bool) {
	sc, ok := ctx.Value(skillContextKey).(*SkillContext)
	return sc, ok
}

// MustGetSkillContext retrieves the SkillContext or panics if not found
func MustGetSkillContext(ctx context.Context) *SkillContext {
	sc, ok := GetSkillContext(ctx)
	if !ok {
		panic("skill context not found in context")
	}
	return sc
}

// GetSkillPath is a convenience function to get the skill path from context
func GetSkillPath(ctx context.Context) string {
	if sc, ok := GetSkillContext(ctx); ok {
		return sc.SkillPath
	}
	return ""
}

// GetSkillName is a convenience function to get the skill name from context
func GetSkillName(ctx context.Context) string {
	if sc, ok := GetSkillContext(ctx); ok {
		return sc.SkillName
	}
	return ""
}
