package google

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go_lib/chatmodel-routing/adapter"

	"github.com/google/uuid"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared/constant"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const defaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"
const thoughtSignatureSeparator = ":::SIG:::"

// Provider implements the adapter.Provider interface for Google Gemini.
// Uses the native Gemini API with proper format conversion.
type Provider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// New creates a new Google Gemini provider with the given API key.
func New(apiKey string) *Provider {
	return &Provider{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{},
	}
}

// DefaultBaseURL returns the default base URL for Google Gemini.
func (p *Provider) DefaultBaseURL() string {
	return defaultBaseURL
}

// GetBaseURL returns the current base URL.
func (p *Provider) GetBaseURL() string {
	return p.baseURL
}

// SetBaseURL sets a custom base URL for the provider.
func (p *Provider) SetBaseURL(url string) {
	p.baseURL = url
}

// SetHTTPClient sets a custom HTTP client for the provider.
func (p *Provider) SetHTTPClient(client *http.Client) {
	p.httpClient = client
}

// Ensure Provider implements ProviderWithBaseURL interface.
var _ adapter.ProviderWithBaseURL = (*Provider)(nil)

// ChatCompletion handles a non-streaming chat completion request.
func (p *Provider) ChatCompletion(ctx context.Context, req *openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	return p.ChatCompletionRaw(ctx, body)
}

// ChatCompletionRaw handles a non-streaming chat completion request with raw JSON body.
func (p *Provider) ChatCompletionRaw(ctx context.Context, body []byte) (*openai.ChatCompletion, error) {
	geminiReq, model, err := p.convertRequestRaw(body)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, model, p.apiKey)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(geminiReq))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google gemini api error: %d - %s", resp.StatusCode, string(respBody))
	}

	// Debug: Print raw response
	// fmt.Printf("DEBUG Gemini Response: %s\n", string(respBody))

	return p.convertResponse(respBody, model)
}

// ChatCompletionStream handles a streaming chat completion request.
func (p *Provider) ChatCompletionStream(ctx context.Context, req *openai.ChatCompletionNewParams) (adapter.Stream, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	return p.ChatCompletionStreamRaw(ctx, body)
}

// ChatCompletionStreamRaw handles a streaming chat completion request with raw JSON body.
func (p *Provider) ChatCompletionStreamRaw(ctx context.Context, body []byte) (adapter.Stream, error) {
	geminiReq, model, err := p.convertRequestRaw(body)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", p.baseURL, model, p.apiKey)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(geminiReq))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("google gemini api error: %d - %s", resp.StatusCode, string(respBody))
	}

	return &geminiStream{
		reader: bufio.NewReader(resp.Body),
		body:   resp.Body,
		model:  model,
		id:     "chatcmpl-" + uuid.New().String()[:8],
	}, nil
}

// convertRequestRaw converts OpenAI format request to Gemini format.
// Returns the converted request body and the model name.
func (p *Provider) convertRequestRaw(body []byte) ([]byte, string, error) {
	parsed := gjson.ParseBytes(body)
	model := parsed.Get("model").String()

	// Build Gemini request
	geminiReq := map[string]interface{}{}

	// Convert messages to contents
	contents := p.convertMessages(parsed.Get("messages").Array())
	geminiReq["contents"] = contents

	// Convert generation config
	genConfig := map[string]interface{}{}
	if temp := parsed.Get("temperature"); temp.Exists() {
		genConfig["temperature"] = temp.Float()
	}
	if topP := parsed.Get("top_p"); topP.Exists() {
		genConfig["topP"] = topP.Float()
	}
	if maxTokens := parsed.Get("max_tokens"); maxTokens.Exists() {
		genConfig["maxOutputTokens"] = maxTokens.Int()
	} else {
		genConfig["maxOutputTokens"] = adapter.GetModelMaxOutputTokens(model)
	}
	if stop := parsed.Get("stop"); stop.Exists() {
		if stop.IsArray() {
			var stops []string
			for _, s := range stop.Array() {
				stops = append(stops, s.String())
			}
			genConfig["stopSequences"] = stops
		} else {
			genConfig["stopSequences"] = []string{stop.String()}
		}
	}
	if len(genConfig) > 0 {
		geminiReq["generationConfig"] = genConfig
	}

	// Convert tools
	if tools := parsed.Get("tools"); tools.Exists() && tools.IsArray() {
		geminiTools := p.convertTools(tools.Array())
		if len(geminiTools) > 0 {
			geminiReq["tools"] = geminiTools
		}
	}

	// Extract system instruction from messages
	if sysInstr := p.extractSystemInstruction(parsed.Get("messages").Array()); sysInstr != "" {
		geminiReq["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": sysInstr},
			},
		}
	}

	result, err := json.Marshal(geminiReq)
	return result, model, err
}

// convertMessages converts OpenAI messages to Gemini contents format.
func (p *Provider) convertMessages(messages []gjson.Result) []map[string]interface{} {
	var contents []map[string]interface{}
	toolIDToName := make(map[string]string)

	for _, msg := range messages {
		role := msg.Get("role").String()

		// Skip system messages (handled separately as systemInstruction)
		if role == "system" {
			continue
		}

		// Extract tool calls from assistant messages to map IDs to names
		if role == "assistant" {
			if toolCalls := msg.Get("tool_calls"); toolCalls.Exists() && toolCalls.IsArray() {
				for _, tc := range toolCalls.Array() {
					id := tc.Get("id").String()
					name := tc.Get("function.name").String()

					// Decode ID if it contains signature (OpenAI -> Gemini direction for previous assistant messages)
					// Note: ID in tool_calls array of assistant message comes from US (Client),
					// which comes from OUR previous response.
					// So it might have the signature encoded.
					if strings.Contains(id, thoughtSignatureSeparator) {
						parts := strings.Split(id, thoughtSignatureSeparator)
						if len(parts) >= 1 {
							id = parts[0]
						}
					}

					if id != "" && name != "" {
						toolIDToName[id] = name
					}
				}
			}
		}

		content := map[string]interface{}{}

		// Map OpenAI roles to Gemini roles
		switch role {
		case "assistant":
			content["role"] = "model"
		case "tool":
			content["role"] = "function"
		default:
			content["role"] = role
		}

		parts := p.convertMessageParts(msg, toolIDToName)
		if len(parts) > 0 {
			content["parts"] = parts
			contents = append(contents, content)
		}
	}

	return contents
}

// convertMessageParts converts message content to Gemini parts format.
func (p *Provider) convertMessageParts(msg gjson.Result, toolIDToName map[string]string) []map[string]interface{} {
	var parts []map[string]interface{}
	role := msg.Get("role").String()

	// Handle tool/function response
	if role == "tool" {
		toolCallID := msg.Get("tool_call_id").String()
		responseContent := msg.Get("content").String()

		// Try to parse content as JSON
		var responseData interface{}
		if err := json.Unmarshal([]byte(responseContent), &responseData); err != nil {
			responseData = map[string]interface{}{"result": responseContent}
		}

		// Resolve function name from tool_call_id
		funcName := toolCallID
		// Clean toolCallID if it contains signature (from Client -> US)
		originalID := toolCallID
		if strings.Contains(toolCallID, thoughtSignatureSeparator) {
			parts := strings.Split(toolCallID, thoughtSignatureSeparator)
			if len(parts) >= 1 {
				originalID = parts[0]
			}
		}

		if name, ok := toolIDToName[originalID]; ok {
			funcName = name
		}

		parts = append(parts, map[string]interface{}{
			"functionResponse": map[string]interface{}{
				"name": funcName,
				"response": map[string]interface{}{
					"content": responseData,
				},
			},
		})
		return parts
	}

	// Handle assistant message with tool calls
	if role == "assistant" {
		toolCalls := msg.Get("tool_calls")
		if toolCalls.Exists() && toolCalls.IsArray() {
			for _, tc := range toolCalls.Array() {
				funcName := tc.Get("function.name").String()
				funcArgs := tc.Get("function.arguments").String()
				toolCallID := tc.Get("id").String()

				var args map[string]interface{}
				if err := json.Unmarshal([]byte(funcArgs), &args); err != nil {
					args = map[string]interface{}{}
				}

				funcCall := map[string]interface{}{
					"name": funcName,
					"args": args,
				}

				// Extract thought_signature from ID or thought_signature field
				var thoughtSig string
				if strings.Contains(toolCallID, thoughtSignatureSeparator) {
					parts := strings.Split(toolCallID, thoughtSignatureSeparator)
					if len(parts) >= 2 {
						thoughtSig = parts[1]
					}
				} else if ts := tc.Get("thought_signature"); ts.Exists() {
					// Fallback if client somehow preserved it in JSON (unlikely for openai-go)
					thoughtSig = ts.String()
				}

				part := map[string]interface{}{
					"functionCall": funcCall,
				}

				// thought_signature must be a sibling of functionCall
				if thoughtSig != "" {
					part["thoughtSignature"] = thoughtSig
				}

				parts = append(parts, part)
			}
		}

		// Also include text content if present
		if content := msg.Get("content"); content.Exists() && content.String() != "" {
			parts = append(parts, map[string]interface{}{
				"text": content.String(),
			})
		}

		return parts
	}

	// Handle regular content (string or array)
	content := msg.Get("content")
	if content.IsArray() {
		for _, part := range content.Array() {
			partType := part.Get("type").String()
			switch partType {
			case "text":
				parts = append(parts, map[string]interface{}{
					"text": part.Get("text").String(),
				})
			case "image_url":
				imageURL := part.Get("image_url.url").String()
				// Handle base64 images
				if strings.HasPrefix(imageURL, "data:") {
					// Parse data URL: data:image/png;base64,xxxx
					dataParts := strings.SplitN(imageURL, ",", 2)
					if len(dataParts) == 2 {
						mimeType := strings.TrimPrefix(strings.Split(dataParts[0], ";")[0], "data:")
						parts = append(parts, map[string]interface{}{
							"inlineData": map[string]interface{}{
								"mimeType": mimeType,
								"data":     dataParts[1],
							},
						})
					}
				} else {
					// External URL
					parts = append(parts, map[string]interface{}{
						"fileData": map[string]interface{}{
							"mimeType": "image/jpeg",
							"fileUri":  imageURL,
						},
					})
				}
			}
		}
	} else if content.Exists() {
		parts = append(parts, map[string]interface{}{
			"text": content.String(),
		})
	}

	return parts
}

// convertTools converts OpenAI tools format to Gemini format.
func (p *Provider) convertTools(tools []gjson.Result) []map[string]interface{} {
	var functionDeclarations []map[string]interface{}

	for _, tool := range tools {
		if tool.Get("type").String() != "function" {
			continue
		}

		funcDef := tool.Get("function")
		funcDecl := map[string]interface{}{
			"name":        funcDef.Get("name").String(),
			"description": funcDef.Get("description").String(),
		}

		if params := funcDef.Get("parameters"); params.Exists() {
			// Convert parameters schema
			funcDecl["parameters"] = p.convertParameterSchema(params)
		}

		functionDeclarations = append(functionDeclarations, funcDecl)
	}

	if len(functionDeclarations) == 0 {
		return nil
	}

	return []map[string]interface{}{
		{"functionDeclarations": functionDeclarations},
	}
}

// convertParameterSchema converts OpenAI JSON Schema to Gemini format.
func (p *Provider) convertParameterSchema(params gjson.Result) map[string]interface{} {
	schema := map[string]interface{}{}

	if t := params.Get("type"); t.Exists() {
		schema["type"] = strings.ToUpper(t.String())
	}

	if desc := params.Get("description"); desc.Exists() {
		schema["description"] = desc.String()
	}

	if props := params.Get("properties"); props.Exists() {
		properties := map[string]interface{}{}
		props.ForEach(func(key, value gjson.Result) bool {
			properties[key.String()] = p.convertParameterSchema(value)
			return true
		})
		schema["properties"] = properties
	}

	if req := params.Get("required"); req.Exists() && req.IsArray() {
		var required []string
		for _, r := range req.Array() {
			required = append(required, r.String())
		}
		schema["required"] = required
	}

	if items := params.Get("items"); items.Exists() {
		schema["items"] = p.convertParameterSchema(items)
	}

	if enum := params.Get("enum"); enum.Exists() && enum.IsArray() {
		var enumVals []string
		for _, e := range enum.Array() {
			enumVals = append(enumVals, e.String())
		}
		schema["enum"] = enumVals
	}

	return schema
}

// extractSystemInstruction extracts system messages and combines them.
func (p *Provider) extractSystemInstruction(messages []gjson.Result) string {
	var systemParts []string
	for _, msg := range messages {
		if msg.Get("role").String() == "system" {
			content := msg.Get("content")
			if content.IsArray() {
				for _, part := range content.Array() {
					if part.Get("type").String() == "text" {
						systemParts = append(systemParts, part.Get("text").String())
					}
				}
			} else if content.Exists() {
				systemParts = append(systemParts, content.String())
			}
		}
	}
	return strings.Join(systemParts, "\n")
}

// convertResponse converts Gemini response to OpenAI format.
func (p *Provider) convertResponse(body []byte, model string) (*openai.ChatCompletion, error) {
	parsed := gjson.ParseBytes(body)

	// Check for error in response
	if errMsg := parsed.Get("error.message"); errMsg.Exists() {
		return nil, fmt.Errorf("gemini error: %s", errMsg.String())
	}

	candidates := parsed.Get("candidates").Array()
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := candidates[0]
	content := candidate.Get("content")
	parts := content.Get("parts").Array()

	var choices []openai.ChatCompletionChoice

	choice := openai.ChatCompletionChoice{
		Index:        0,
		FinishReason: p.convertFinishReason(candidate.Get("finishReason").String()),
	}

	// Build message
	message := openai.ChatCompletionMessage{
		Role: constant.ValueOf[constant.Assistant](),
	}

	var textParts []string
	var reasoningParts []string
	var toolCalls []openai.ChatCompletionMessageToolCall

	for i, part := range parts {
		// Try to find text content and check for reasoning
		var content string
		isReasoning := false

		if text := part.Get("text"); text.Exists() {
			content = text.String()
			// Check if it's marked as thought
			if thought := part.Get("thought"); (thought.Exists() && thought.Bool()) || (part.Get("role").String() == "thought") {
				isReasoning = true
			}
		} else if thought := part.Get("thought"); thought.Exists() && thought.Type == gjson.String {
			// Handle case where "thought" is the content field
			content = thought.String()
			isReasoning = true
		}

		if content != "" {
			if isReasoning {
				reasoningParts = append(reasoningParts, content)
			} else {
				textParts = append(textParts, content)
			}
		}

		if funcCall := part.Get("functionCall"); funcCall.Exists() {
			// DEBUG: Print funcCall
			// fmt.Printf("DEBUG: funcCall raw: %s\n", funcCall.String())

			funcName := funcCall.Get("name").String()
			funcArgs := funcCall.Get("args")

			// Try both camelCase and snake_case for thought signature
			// Note: thoughtSignature is a sibling of functionCall in the part object, not a child
			thoughtSig := part.Get("thoughtSignature").String()
			if thoughtSig == "" {
				thoughtSig = part.Get("thought_signature").String()
			}

			// Fallback: Check inside functionCall just in case (e.g. TestConvertResponse_Logic mock data)
			if thoughtSig == "" {
				thoughtSig = funcCall.Get("thoughtSignature").String()
			}
			if thoughtSig == "" {
				thoughtSig = funcCall.Get("thought_signature").String()
			}

			argsJSON, _ := json.Marshal(funcArgs.Value())

			toolCall := openai.ChatCompletionMessageToolCall{
				ID:   fmt.Sprintf("call_%d", i),
				Type: "function",
				Function: openai.ChatCompletionMessageToolCallFunction{
					Name:      funcName,
					Arguments: string(argsJSON),
				},
			}

			// Store thought_signature in the tool call for later retrieval
			if thoughtSig != "" {
				// Encode thought_signature into the ID so it survives the round-trip through strict clients/structs
				toolCall.ID = toolCall.ID + thoughtSignatureSeparator + thoughtSig
			}

			toolCalls = append(toolCalls, toolCall)
		}
	}

	if len(textParts) > 0 {
		message.Content = strings.Join(textParts, "")
	}

	if len(toolCalls) > 0 {
		message.ToolCalls = toolCalls
	}

	choice.Message = message
	choices = append(choices, choice)

	// Build usage
	usage := openai.CompletionUsage{}
	if usageData := parsed.Get("usageMetadata"); usageData.Exists() {
		usage.PromptTokens = usageData.Get("promptTokenCount").Int()
		usage.CompletionTokens = usageData.Get("candidatesTokenCount").Int()
		usage.TotalTokens = usageData.Get("totalTokenCount").Int()
	}

	result := &openai.ChatCompletion{
		ID:      "chatcmpl-" + uuid.New().String()[:8],
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: choices,
		Usage:   usage,
	}

	// Post-processing for fields not in openai-go struct (reasoning_content, thought_signature)
	// We check if we need to modify the JSON
	needsJSONMod := len(reasoningParts) > 0
	if !needsJSONMod {
		for _, part := range parts {
			if part.Get("functionCall.thoughtSignature").Exists() {
				needsJSONMod = true
				break
			}
		}
	}

	if needsJSONMod {
		resultJSON, _ := json.Marshal(result)

		if len(reasoningParts) > 0 {
			reasoningContent := strings.Join(reasoningParts, "\n")
			resultJSON, _ = sjson.SetBytes(resultJSON, "choices.0.message.reasoning_content", reasoningContent)
		}

		// We encoded thought_signature in ID, so we don't strictly need to add it to JSON as a separate field
		// unless we want to be nice to clients that inspect JSON.
		// However, ID encoding is the primary persistence mechanism.

		json.Unmarshal(resultJSON, result)
	}

	return result, nil
}

// convertFinishReason converts Gemini finish reason to OpenAI format.
func (p *Provider) convertFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY":
		return "content_filter"
	case "RECITATION":
		return "content_filter"
	case "TOOL_CODE":
		return "tool_calls"
	default:
		return "stop"
	}
}

// geminiStream implements the adapter.Stream interface for Gemini streaming.
type geminiStream struct {
	reader        *bufio.Reader
	body          io.ReadCloser
	model         string
	id            string
	toolCallIndex int
}

func (s *geminiStream) Recv() (*openai.ChatCompletionChunk, error) {
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil, io.EOF
			}
			return nil, err
		}

		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse SSE data
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Parse the Gemini response chunk
		parsed := gjson.Parse(data)

		// Check for error
		if errMsg := parsed.Get("error.message"); errMsg.Exists() {
			return nil, fmt.Errorf("gemini stream error: %s", errMsg.String())
		}

		candidates := parsed.Get("candidates").Array()
		if len(candidates) == 0 {
			continue
		}

		candidate := candidates[0]
		parts := candidate.Get("content.parts").Array()

		chunk := &openai.ChatCompletionChunk{
			ID:      s.id,
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   s.model,
		}

		delta := openai.ChatCompletionChunkChoiceDelta{}
		var toolCalls []openai.ChatCompletionChunkChoiceDeltaToolCall

		for _, part := range parts {
			// Try to find text content and check for reasoning
			var content string

			if text := part.Get("text"); text.Exists() {
				content = text.String()
				// We currently treat reasoning as content in streaming
				// because we can't easily inject ReasoningContent into the struct/JSON
				// without potentially breaking the stream or losing data if the struct doesn't support it.
			} else if thought := part.Get("thought"); thought.Exists() && thought.Type == gjson.String {
				content = thought.String()
			}

			if content != "" {
				delta.Content += content
			}

			if funcCall := part.Get("functionCall"); funcCall.Exists() {
				funcName := funcCall.Get("name").String()
				funcArgs := funcCall.Get("args")
				argsJSON, _ := json.Marshal(funcArgs.Value())

				toolCall := openai.ChatCompletionChunkChoiceDeltaToolCall{
					Index: int64(s.toolCallIndex),
					ID:    fmt.Sprintf("call_%d", s.toolCallIndex),
					Type:  "function",
					Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
						Name:      funcName,
						Arguments: string(argsJSON),
					},
				}

				if thoughtSig := part.Get("functionCall.thoughtSignature").String(); thoughtSig != "" {
					toolCall.ID = toolCall.ID + thoughtSignatureSeparator + thoughtSig
				} else if thoughtSig := part.Get("functionCall.thought_signature").String(); thoughtSig != "" {
					toolCall.ID = toolCall.ID + thoughtSignatureSeparator + thoughtSig
				} else if thoughtSig := part.Get("thoughtSignature").String(); thoughtSig != "" {
					toolCall.ID = toolCall.ID + thoughtSignatureSeparator + thoughtSig
				} else if thoughtSig := part.Get("thought_signature").String(); thoughtSig != "" {
					toolCall.ID = toolCall.ID + thoughtSignatureSeparator + thoughtSig
				}

				toolCalls = append(toolCalls, toolCall)
				s.toolCallIndex++
			}
		}

		if len(toolCalls) > 0 {
			delta.ToolCalls = toolCalls
		}

		finishReason := ""
		if fr := candidate.Get("finishReason"); fr.Exists() {
			finishReason = convertStreamFinishReason(fr.String())
		}

		choice := openai.ChatCompletionChunkChoice{
			Index: 0,
			Delta: delta,
		}
		if finishReason != "" {
			choice.FinishReason = finishReason
		}

		chunk.Choices = []openai.ChatCompletionChunkChoice{choice}

		// Include usage if present
		if usageData := parsed.Get("usageMetadata"); usageData.Exists() {
			chunk.Usage = openai.CompletionUsage{
				PromptTokens:     usageData.Get("promptTokenCount").Int(),
				CompletionTokens: usageData.Get("candidatesTokenCount").Int(),
				TotalTokens:      usageData.Get("totalTokenCount").Int(),
			}
		}

		return chunk, nil
	}
}

func (s *geminiStream) Close() error {
	return s.body.Close()
}

func convertStreamFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY":
		return "content_filter"
	case "RECITATION":
		return "content_filter"
	case "TOOL_CODE":
		return "tool_calls"
	default:
		return ""
	}
}
