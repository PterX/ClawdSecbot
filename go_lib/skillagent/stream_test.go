package skillagent

import (
	"errors"
	"testing"
	"time"
)

func TestStreamEventTypeString(t *testing.T) {
	tests := []struct {
		eventType StreamEventType
		want      string
	}{
		{StreamEventSkillDiscovery, "skill_discovery"},
		{StreamEventSkillSelected, "skill_selected"},
		{StreamEventSkillActivated, "skill_activated"},
		{StreamEventToolCalling, "tool_calling"},
		{StreamEventToolResult, "tool_result"},
		{StreamEventAgentThinking, "agent_thinking"},
		{StreamEventPartialOutput, "partial_output"},
		{StreamEventFinalOutput, "final_output"},
		{StreamEventError, "error"},
		{StreamEventComplete, "complete"},
		{StreamEventType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.eventType.String(); got != tt.want {
				t.Fatalf("Expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestStreamEventEmitterSkipsWhenChannelIsFull(t *testing.T) {
	ch := make(chan StreamEvent, 1)
	emitter := NewStreamEventEmitter(ch)

	emitter.EmitPartialOutput("first")
	emitter.EmitPartialOutput("second")

	event := <-ch
	if event.Type != StreamEventPartialOutput {
		t.Fatalf("Expected partial output event, got %s", event.Type.String())
	}
	if data, ok := event.Data.(PartialOutputData); !ok || data.Content != "first" {
		t.Fatalf("Expected first partial output payload, got %#v", event.Data)
	}
	select {
	case unexpected := <-ch:
		t.Fatalf("Expected full channel to skip second event, got %#v", unexpected)
	default:
	}
}

func TestStreamEventEmitterCloseStopsEmitting(t *testing.T) {
	ch := make(chan StreamEvent, 1)
	emitter := NewStreamEventEmitter(ch)
	emitter.Close()

	emitter.EmitError(errors.New("boom"))

	select {
	case event := <-ch:
		t.Fatalf("Expected closed emitter to skip events, got %#v", event)
	default:
	}
}

func TestStreamEventEmitterToolResultIncludesErrorText(t *testing.T) {
	ch := make(chan StreamEvent, 1)
	emitter := NewStreamEventEmitter(ch)

	emitter.EmitToolResult("read_file", "partial", time.Second, errors.New("denied"))

	event := <-ch
	if event.Type != StreamEventToolResult {
		t.Fatalf("Expected tool result event, got %s", event.Type.String())
	}
	data, ok := event.Data.(ToolResultData)
	if !ok {
		t.Fatalf("Expected ToolResultData, got %#v", event.Data)
	}
	if data.Error != "denied" {
		t.Fatalf("Expected error text to be recorded, got %q", data.Error)
	}
}
