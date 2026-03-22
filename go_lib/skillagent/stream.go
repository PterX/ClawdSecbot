package skillagent

import (
	"time"
)

// StreamEventType represents the type of stream event
type StreamEventType int

const (
	// StreamEventSkillDiscovery indicates skills are being discovered
	StreamEventSkillDiscovery StreamEventType = iota
	// StreamEventSkillSelected indicates a skill has been selected
	StreamEventSkillSelected
	// StreamEventSkillActivated indicates a skill has been activated
	StreamEventSkillActivated
	// StreamEventToolCalling indicates a tool is being called
	StreamEventToolCalling
	// StreamEventToolResult indicates a tool has returned a result
	StreamEventToolResult
	// StreamEventAgentThinking indicates the agent is thinking/reasoning
	StreamEventAgentThinking
	// StreamEventPartialOutput indicates partial output is available
	StreamEventPartialOutput
	// StreamEventFinalOutput indicates the final output is ready
	StreamEventFinalOutput
	// StreamEventError indicates an error occurred
	StreamEventError
	// StreamEventComplete indicates execution is complete
	StreamEventComplete
)

// String returns the string representation of the event type
func (t StreamEventType) String() string {
	switch t {
	case StreamEventSkillDiscovery:
		return "skill_discovery"
	case StreamEventSkillSelected:
		return "skill_selected"
	case StreamEventSkillActivated:
		return "skill_activated"
	case StreamEventToolCalling:
		return "tool_calling"
	case StreamEventToolResult:
		return "tool_result"
	case StreamEventAgentThinking:
		return "agent_thinking"
	case StreamEventPartialOutput:
		return "partial_output"
	case StreamEventFinalOutput:
		return "final_output"
	case StreamEventError:
		return "error"
	case StreamEventComplete:
		return "complete"
	default:
		return "unknown"
	}
}

// StreamEvent represents an event during streaming execution
type StreamEvent struct {
	// Type is the type of event
	Type StreamEventType `json:"type"`

	// Timestamp is when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// Data contains event-specific data
	Data interface{} `json:"data,omitempty"`

	// Error contains error information if Type is StreamEventError
	Error error `json:"error,omitempty"`
}

// SkillDiscoveryData contains data for skill discovery events
type SkillDiscoveryData struct {
	SkillCount int              `json:"skill_count"`
	Skills     []*SkillMetadata `json:"skills"`
}

// SkillSelectedData contains data for skill selection events
type SkillSelectedData struct {
	SkillName   string `json:"skill_name"`
	Description string `json:"description"`
}

// SkillActivatedData contains data for skill activation events
type SkillActivatedData struct {
	SkillName    string   `json:"skill_name"`
	Version      string   `json:"version,omitempty"`
	AllowedTools []string `json:"allowed_tools,omitempty"`
}

// ToolCallingData contains data for tool calling events
type ToolCallingData struct {
	ToolName  string `json:"tool_name"`
	Arguments string `json:"arguments"`
}

// ToolResultData contains data for tool result events
type ToolResultData struct {
	ToolName string        `json:"tool_name"`
	Result   string        `json:"result"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

// AgentThinkingData contains data for agent thinking events
type AgentThinkingData struct {
	Thought string `json:"thought"`
}

// PartialOutputData contains data for partial output events
type PartialOutputData struct {
	Content string `json:"content"`
}

// FinalOutputData contains data for final output events
type FinalOutputData struct {
	Content string `json:"content"`
}

// CompleteData contains data for completion events
type CompleteData struct {
	Success       bool          `json:"success"`
	Duration      time.Duration `json:"duration"`
	ToolCallCount int           `json:"tool_call_count"`
}

// NewStreamEvent creates a new StreamEvent
func NewStreamEvent(eventType StreamEventType, data interface{}) StreamEvent {
	return StreamEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// NewErrorEvent creates a new error StreamEvent
func NewErrorEvent(err error) StreamEvent {
	return StreamEvent{
		Type:      StreamEventError,
		Timestamp: time.Now(),
		Error:     err,
	}
}

// StreamEventEmitter helps emit events to a channel
type StreamEventEmitter struct {
	ch        chan<- StreamEvent
	closed    bool
	startTime time.Time
}

// NewStreamEventEmitter creates a new StreamEventEmitter
func NewStreamEventEmitter(ch chan<- StreamEvent) *StreamEventEmitter {
	return &StreamEventEmitter{
		ch:        ch,
		startTime: time.Now(),
	}
}

// Emit sends an event to the channel
func (e *StreamEventEmitter) Emit(event StreamEvent) {
	if e.closed || e.ch == nil {
		return
	}
	select {
	case e.ch <- event:
	default:
		// Channel full, skip event
	}
}

// EmitSkillDiscovery emits a skill discovery event
func (e *StreamEventEmitter) EmitSkillDiscovery(skills []*SkillMetadata) {
	e.Emit(NewStreamEvent(StreamEventSkillDiscovery, SkillDiscoveryData{
		SkillCount: len(skills),
		Skills:     skills,
	}))
}

// EmitSkillSelected emits a skill selected event
func (e *StreamEventEmitter) EmitSkillSelected(skillName, description string) {
	e.Emit(NewStreamEvent(StreamEventSkillSelected, SkillSelectedData{
		SkillName:   skillName,
		Description: description,
	}))
}

// EmitSkillActivated emits a skill activated event
func (e *StreamEventEmitter) EmitSkillActivated(manifest *SkillManifest) {
	e.Emit(NewStreamEvent(StreamEventSkillActivated, SkillActivatedData{
		SkillName:    manifest.Name,
		Version:      manifest.Version,
		AllowedTools: manifest.AllowedTools,
	}))
}

// EmitToolCalling emits a tool calling event
func (e *StreamEventEmitter) EmitToolCalling(toolName, arguments string) {
	e.Emit(NewStreamEvent(StreamEventToolCalling, ToolCallingData{
		ToolName:  toolName,
		Arguments: arguments,
	}))
}

// EmitToolResult emits a tool result event
func (e *StreamEventEmitter) EmitToolResult(toolName, result string, duration time.Duration, err error) {
	data := ToolResultData{
		ToolName: toolName,
		Result:   result,
		Duration: duration,
	}
	if err != nil {
		data.Error = err.Error()
	}
	e.Emit(NewStreamEvent(StreamEventToolResult, data))
}

// EmitAgentThinking emits an agent thinking event
func (e *StreamEventEmitter) EmitAgentThinking(thought string) {
	e.Emit(NewStreamEvent(StreamEventAgentThinking, AgentThinkingData{
		Thought: thought,
	}))
}

// EmitPartialOutput emits a partial output event
func (e *StreamEventEmitter) EmitPartialOutput(content string) {
	e.Emit(NewStreamEvent(StreamEventPartialOutput, PartialOutputData{
		Content: content,
	}))
}

// EmitFinalOutput emits a final output event
func (e *StreamEventEmitter) EmitFinalOutput(content string) {
	e.Emit(NewStreamEvent(StreamEventFinalOutput, FinalOutputData{
		Content: content,
	}))
}

// EmitError emits an error event
func (e *StreamEventEmitter) EmitError(err error) {
	e.Emit(NewErrorEvent(err))
}

// EmitComplete emits a completion event and closes the emitter
func (e *StreamEventEmitter) EmitComplete(success bool, toolCallCount int) {
	e.Emit(NewStreamEvent(StreamEventComplete, CompleteData{
		Success:       success,
		Duration:      time.Since(e.startTime),
		ToolCallCount: toolCallCount,
	}))
}

// Close marks the emitter as closed
func (e *StreamEventEmitter) Close() {
	e.closed = true
}
