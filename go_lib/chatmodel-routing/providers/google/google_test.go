package google

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

// User provided config
const (
	testAPIKey = ""
	testModel  = "gemini-3-pro-preview"
	// Using the model specified by user.
)

func TestGoogleProvider_Integration_Raw(t *testing.T) {
	if testAPIKey == "" {
		t.Skip("Skipping integration test: API key not provided")
	}

	p := New(testAPIKey)
	ctx := context.Background()

	// 1. Test Non-Streaming Text & Reasoning
	t.Run("Text_Reasoning_NonStream", func(t *testing.T) {
		reqMap := map[string]interface{}{
			"model": testModel,
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": "How many rs in strawberry?",
				},
			},
		}
		reqBytes, _ := json.Marshal(reqMap)

		resp, err := p.ChatCompletionRaw(ctx, reqBytes)
		if err != nil {
			t.Fatalf("ChatCompletionRaw failed: %v", err)
		}

		if len(resp.Choices) == 0 {
			t.Fatal("No choices returned")
		}

		content := resp.Choices[0].Message.Content
		t.Logf("Content: %s", content)

		// Check for reasoning content (injected via JSON hack in google.go)
		respJSON, _ := json.Marshal(resp)
		reasoning := gjson.GetBytes(respJSON, "choices.0.message.reasoning_content").String()
		if reasoning != "" {
			t.Logf("Reasoning extracted: %s", reasoning)
		} else {
			t.Log("No reasoning content found (might be model behavior or parsing issue)")
		}
	})

	// 2. Test Streaming Text & Reasoning
	t.Run("Text_Reasoning_Stream", func(t *testing.T) {
		reqMap := map[string]interface{}{
			"model": testModel,
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": "Explain quantum computing briefly.",
				},
			},
			"stream": true,
		}
		reqBytes, _ := json.Marshal(reqMap)

		stream, err := p.ChatCompletionStreamRaw(ctx, reqBytes)
		if err != nil {
			t.Fatalf("ChatCompletionStreamRaw failed: %v", err)
		}
		defer stream.Close()

		var fullContent string
		for {
			chunk, err := stream.Recv()
			if err != nil {
				if strings.Contains(err.Error(), "EOF") {
					break
				}
				t.Fatalf("Stream recv failed: %v", err)
			}

			if len(chunk.Choices) > 0 {
				fullContent += chunk.Choices[0].Delta.Content
			}
		}

		if fullContent == "" {
			t.Error("Empty stream content")
		}
		t.Logf("Stream full content: %s", fullContent)
	})

	// 3. Test Tool Calling (with signature handling)
	t.Run("Tool_Calling_RoundTrip", func(t *testing.T) {
		// Define tool
		toolDef := map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_current_weather",
				"description": "Get the current weather in a given location",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type": "string",
						},
					},
					"required": []string{"location"},
				},
			},
		}

		// Step 1: User asks question
		messages := []map[string]interface{}{
			{
				"role":    "user",
				"content": "What's the weather in Tokyo?",
			},
		}

		req1Map := map[string]interface{}{
			"model":    testModel,
			"messages": messages,
			"tools":    []interface{}{toolDef},
		}
		req1Bytes, _ := json.Marshal(req1Map)

		// Call 1: Get Tool Call
		resp1, err := p.ChatCompletionRaw(ctx, req1Bytes)
		if err != nil {
			t.Fatalf("Step 1 failed: %v", err)
		}

		msg1 := resp1.Choices[0].Message
		if len(msg1.ToolCalls) == 0 {
			t.Skip("Model didn't call tool, skipping tool test")
		}

		toolCall := msg1.ToolCalls[0]
		t.Logf("Tool Call ID: %s", toolCall.ID)
		t.Logf("Function: %s, Args: %s", toolCall.Function.Name, toolCall.Function.Arguments)

		// Verify ID contains signature if model supports it
		if strings.Contains(toolCall.ID, ":::SIG:::") {
			t.Log("Successfully captured thought signature in ID!")
		} else {
			t.Log("No thought signature in ID (model might not support it or it wasn't returned)")
		}

		// Step 2: Send Tool Result back
		// Add assistant message (need to reconstruct manually because ToolCalls struct is from openai package)
		// We can reuse the `msg1` content but we need to marshal it to map to append to messages

		// Reconstruct tool calls for the next request map
		toolCallsMap := []map[string]interface{}{}
		for _, tc := range msg1.ToolCalls {
			toolCallsMap = append(toolCallsMap, map[string]interface{}{
				"id":   tc.ID,
				"type": "function",
				"function": map[string]interface{}{
					"name":      tc.Function.Name,
					"arguments": tc.Function.Arguments,
				},
			})
		}

		asstMsg := map[string]interface{}{
			"role":       "assistant",
			"content":    msg1.Content,
			"tool_calls": toolCallsMap,
		}
		messages = append(messages, asstMsg)

		// Add tool message
		toolMsg := map[string]interface{}{
			"role":         "tool",
			"tool_call_id": toolCall.ID,
			"content":      "{\"weather\": \"Sunny\", \"temp\": 25}",
		}
		messages = append(messages, toolMsg)

		req2Map := map[string]interface{}{
			"model":    testModel,
			"messages": messages,
			"tools":    []interface{}{toolDef},
		}
		req2Bytes, _ := json.Marshal(req2Map)

		// Call 2: Get Final Answer
		resp2, err := p.ChatCompletionRaw(ctx, req2Bytes)
		if err != nil {
			t.Fatalf("Step 2 failed: %v", err)
		}

		t.Logf("Final response: %s", resp2.Choices[0].Message.Content)
	})
}

// Unit Tests for Logic Verification (No API Call)
func TestConvertMessages_Logic(t *testing.T) {
	p := New("test")

	// Test Tool Call ID Mapping and Signature Decoding
	msgs := []map[string]interface{}{
		{
			"role":    "user",
			"content": "Hi",
		},
		{
			"role": "assistant",
			"tool_calls": []map[string]interface{}{
				{
					"id":   "call_123:::SIG:::signature_abc",
					"type": "function",
					"function": map[string]interface{}{
						"name":      "my_func",
						"arguments": "{}",
					},
				},
			},
		},
		{
			"role":         "tool",
			"tool_call_id": "call_123:::SIG:::signature_abc",
			"content":      "result",
		},
	}

	msgBytes, _ := json.Marshal(msgs)
	parsed := gjson.ParseBytes(msgBytes)

	contents := p.convertMessages(parsed.Array())

	if len(contents) != 3 {
		t.Fatalf("Expected 3 contents, got %d", len(contents))
	}

	// Check Assistant Message (Index 1) - Signature Extraction
	asstParts := contents[1]["parts"].([]map[string]interface{})
	// funcCall := asstParts[0]["functionCall"].(map[string]interface{}) // No longer inside functionCall
	part := asstParts[0]
	if sig, ok := part["thoughtSignature"]; !ok || sig != "signature_abc" {
		t.Errorf("Expected signature 'signature_abc', got %v", sig)
	}

	// Check Tool Message (Index 2) - Name Mapping
	toolParts := contents[2]["parts"].([]map[string]interface{})
	funcResp := toolParts[0]["functionResponse"].(map[string]interface{})
	if name := funcResp["name"]; name != "my_func" {
		t.Errorf("Expected function name 'my_func', got %v", name)
	}
}

func TestConvertResponse_Logic(t *testing.T) {
	p := New("test")

	// Mock Gemini Response with Signature
	geminiResp := `{
		"candidates": [{
			"content": {
				"parts": [{
					"functionCall": {
						"name": "my_func",
						"args": {},
						"thoughtSignature": "new_sig_123"
					}
				}]
			}
		}]
	}`

	// We need to implement a dummy method or just test the logic indirectly if possible,
	// but convertResponse is exported (capitalized in my memory? No, it's lower case `convertResponse`).
	// Wait, in google.go it is `func (p *Provider) convertResponse(...)`.
	// Since I am in package `google`, I can call it.

	resp, err := p.convertResponse([]byte(geminiResp), "model")
	if err != nil {
		t.Fatalf("convertResponse failed: %v", err)
	}

	id := resp.Choices[0].Message.ToolCalls[0].ID
	if !strings.Contains(id, ":::SIG:::new_sig_123") {
		t.Errorf("Expected ID to contain signature, got %s", id)
	}
}

func TestStreamingID_Logic(t *testing.T) {
	// Need to test geminiStream.Recv logic which adds unique IDs and signature.
	// This is harder to mock without a real reader.
	// I'll skip this for unit test and rely on integration test or logic inspection.
	// But I can create a dummy reader.

	mockBody := `data: {"candidates": [{"content": {"parts": [{"functionCall": {"name": "foo", "args": {}, "thoughtSignature": "sig1"}}]}}]}
`

	stream := &geminiStream{
		reader:        bufio.NewReader(strings.NewReader(mockBody)),
		body:          &dummyCloser{},
		model:         "model",
		id:            "id",
		toolCallIndex: 0,
	}

	chunk, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	if len(chunk.Choices) == 0 {
		t.Fatal("No choices")
	}

	toolCall := chunk.Choices[0].Delta.ToolCalls[0]
	if !strings.Contains(toolCall.ID, ":::SIG:::sig1") {
		t.Errorf("Expected ID to contain signature, got %s", toolCall.ID)
	}
}

type dummyCloser struct{}

func (d *dummyCloser) Close() error                     { return nil }
func (d *dummyCloser) Read(p []byte) (n int, err error) { return 0, io.EOF }
