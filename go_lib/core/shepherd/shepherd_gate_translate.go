package shepherd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go_lib/chatmodel-routing/adapter"
	"go_lib/core/logging"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

const translateSystemPrompt = `You are a precise translator for security warning messages.
Translate the ENTIRE message to match the language of the user's text,
including product names, labels, status values, and instructions.
Preserve the overall structure and formatting (brackets, pipes, newlines).
If the user's language is the same as the message, return the message as-is.
Output ONLY the translated text, no explanations.`

// TranslateForUser translates an English security message to match the user's last input language.
func (sg *ShepherdGate) TranslateForUser(ctx context.Context, message string, lastUserMessage string) string {
	if lastUserMessage == "" {
		logging.Info("[ShepherdGate] TranslateForUser: lastUserMessage is empty, skip translation")
		return message
	}

	sg.mu.RLock()
	chatModel := sg.chatModel
	modelCfg := sg.modelConfig
	sg.mu.RUnlock()

	if chatModel == nil {
		logging.Warning("[ShepherdGate] TranslateForUser: chatModel is nil, returning English message")
		return message
	}

	maxTokens := 1024
	if modelCfg != nil {
		maxTokens = adapter.GetModelMaxOutputTokens(modelCfg.Model)
	}

	sample := lastUserMessage
	if len(sample) > 200 {
		sample = sample[:200]
	}

	logging.Info("[ShepherdGate] TranslateForUser: translating message, userSample=%q", sample)

	userPrompt := fmt.Sprintf("User's language sample: \"%s\"\n\nMessage to translate:\n%s", sample, message)

	resp, err := chatModel.Generate(ctx,
		[]*schema.Message{
			schema.SystemMessage(translateSystemPrompt),
			schema.UserMessage(userPrompt),
		},
		model.WithTemperature(0),
		model.WithMaxTokens(maxTokens),
	)

	if err != nil {
		logging.Warning("[ShepherdGate] TranslateForUser failed, returning English message: %v", err)
		return message
	}

	translated := strings.TrimSpace(resp.Content)
	if translated == "" {
		if rc, ok := resp.Extra["reasoning_content"]; ok {
			if s, ok := rc.(string); ok {
				translated = strings.TrimSpace(s)
			}
		}
	}

	if translated == "" {
		extraKeys := make([]string, 0, len(resp.Extra))
		for k := range resp.Extra {
			extraKeys = append(extraKeys, k)
		}
		extraJSON, _ := json.Marshal(resp.Extra)
		logging.Warning("[ShepherdGate] TranslateForUser returned empty response: Content=%q, Role=%s, ExtraKeys=%v, Extra=%s",
			resp.Content, resp.Role, extraKeys, string(extraJSON))
		return message
	}

	logging.Info("[ShepherdGate] TranslateForUser: translation completed, length=%d", len(translated))
	return translated
}
