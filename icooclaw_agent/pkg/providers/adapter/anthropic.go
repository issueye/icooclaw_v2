package adapter

import (
	"encoding/json"
	"sort"
	"strings"

	"icooclaw/pkg/providers/models/anthropic_model"
)

// ToAnthropicRequest 将中间层请求转换为 Anthropic 请求。
func ToAnthropicRequest(req Request) anthropic_model.Request {
	result := anthropic_model.Request{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream,
		Temperature: req.Temperature,
		Messages:    make([]anthropic_model.Message, 0, len(req.Messages)),
		Tools:       make([]anthropic_model.Tool, 0, len(req.Tools)),
	}

	for _, message := range req.Messages {
		if message.Role == "system" {
			if req.System == "" {
				req.System = message.Content
			}
			continue
		}

		if converted, ok := toAnthropicMessage(message); ok {
			result.Messages = append(result.Messages, converted)
		}
	}

	if req.System != "" {
		result.System = req.System
	}

	for _, tool := range req.Tools {
		result.Tools = append(result.Tools, anthropic_model.Tool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: normalizeAnthropicInputSchema(tool.Parameters),
		})
	}

	return result
}

func toAnthropicMessage(message Message) (anthropic_model.Message, bool) {
	switch message.Role {
	case "user":
		return anthropic_model.Message{
			Role:    "user",
			Content: message.Content,
		}, true
	case "assistant":
		contentBlocks := make([]anthropic_model.ContentBlock, 0, len(message.ToolCalls)+1)
		if strings.TrimSpace(message.Content) != "" {
			contentBlocks = append(contentBlocks, anthropic_model.ContentBlock{
				Type: "text",
				Text: message.Content,
			})
		}
		for _, toolCall := range message.ToolCalls {
			arguments := strings.TrimSpace(toolCall.Arguments)
			if arguments == "" {
				arguments = "{}"
			}
			contentBlocks = append(contentBlocks, anthropic_model.ContentBlock{
				Type:  "tool_use",
				ID:    toolCall.ID,
				Name:  toolCall.Name,
				Input: json.RawMessage(arguments),
			})
		}
		if len(contentBlocks) == 0 {
			return anthropic_model.Message{}, false
		}
		return anthropic_model.Message{
			Role:    "assistant",
			Content: contentBlocks,
		}, true
	case "tool":
		if strings.TrimSpace(message.ToolCallID) == "" {
			return anthropic_model.Message{}, false
		}
		return anthropic_model.Message{
			Role: "user",
			Content: []anthropic_model.ContentBlock{
				{
					Type:      "tool_result",
					ToolUseID: message.ToolCallID,
					Content:   message.Content,
				},
			},
		}, true
	default:
		if strings.TrimSpace(message.Content) == "" {
			return anthropic_model.Message{}, false
		}
		return anthropic_model.Message{
			Role:    "user",
			Content: message.Content,
		}, true
	}
}

func normalizeAnthropicInputSchema(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}

	normalized, ok := normalizeAnthropicSchemaValue(schema).(map[string]any)
	if !ok {
		return schema
	}
	return normalized
}

func normalizeAnthropicSchemaValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return normalizeAnthropicSchemaMap(typed)
	case []any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, normalizeAnthropicSchemaValue(item))
		}
		return items
	default:
		return value
	}
}

func normalizeAnthropicSchemaMap(schema map[string]any) map[string]any {
	normalized := make(map[string]any, len(schema))
	for key, value := range schema {
		normalized[key] = normalizeAnthropicSchemaValue(value)
	}

	properties, ok := normalized["properties"].(map[string]any)
	if !ok {
		return normalized
	}

	requiredSet := map[string]struct{}{}
	if current, ok := normalized["required"].([]any); ok {
		for _, item := range current {
			if name, ok := item.(string); ok && name != "" {
				requiredSet[name] = struct{}{}
			}
		}
	}
	if current, ok := normalized["required"].([]string); ok {
		for _, name := range current {
			if name != "" {
				requiredSet[name] = struct{}{}
			}
		}
	}

	requiredList := make([]string, 0, len(properties))
	for name, propertyValue := range properties {
		property, ok := propertyValue.(map[string]any)
		if !ok {
			continue
		}
		if required, exists := property["required"].(bool); exists {
			delete(property, "required")
			if required {
				requiredSet[name] = struct{}{}
			}
		}
	}

	for name := range requiredSet {
		requiredList = append(requiredList, name)
	}
	if len(requiredList) == 0 {
		delete(normalized, "required")
		return normalized
	}

	sort.Strings(requiredList)
	required := make([]any, 0, len(requiredList))
	for _, name := range requiredList {
		required = append(required, name)
	}
	normalized["required"] = required
	return normalized
}

// FromAnthropicResponse 将 Anthropic 响应转换为中间层响应。
func FromAnthropicResponse(resp anthropic_model.Response) Response {
	result := Response{
		ID:     resp.ID,
		Model:  resp.Model,
		Source: "anthropic",
		Raw:    mustMarshalRaw(resp),
		Usage: Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			result.Content += block.Text
		case "thinking":
			if result.Reasoning != "" && block.Thinking != "" {
				result.Reasoning += "\n"
			}
			result.Reasoning += block.Thinking
		case "tool_use":
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:        block.ID,
				Type:      "function",
				Name:      block.Name,
				Arguments: string(block.Input),
				Source:    "anthropic",
				Raw:       mustMarshalRaw(block),
			})
		}
	}

	result.Content = strings.TrimSpace(result.Content)
	result.Reasoning = strings.TrimSpace(result.Reasoning)
	return result
}

// FromAnthropicEvent 将 Anthropic 流式事件转换为中间层事件。
func FromAnthropicEvent(event anthropic_model.Event) StreamEvent {
	result := StreamEvent{
		Type:   event.Type,
		Source: "anthropic",
		Raw:    mustMarshalRaw(event),
	}

	switch event.Type {
	case "content_block_start":
		if event.ContentBlock.Type == "tool_use" || event.ContentBlock.Name != "" || event.ContentBlock.ID != "" {
			arguments := strings.TrimSpace(string(event.ContentBlock.Input))
			if arguments == "" {
				arguments = "{}"
			}
			result.ToolCalls = []ToolCall{
				{
					ID:        event.ContentBlock.ID,
					Type:      "function",
					Name:      event.ContentBlock.Name,
					Arguments: arguments,
					Source:    "anthropic",
					Raw:       mustMarshalRaw(event),
				},
			}
		}
	case "content_block_delta":
		switch event.Delta.Type {
		case "text_delta":
			result.Content = event.Delta.Text
		case "thinking_delta":
			result.Reasoning = event.Delta.Thinking
		case "input_json_delta":
			if event.ContentBlock.Type == "tool_use" || event.ContentBlock.Name != "" || event.ContentBlock.ID != "" {
				result.ToolCalls = []ToolCall{
					{
						ID:        event.ContentBlock.ID,
						Type:      "function",
						Name:      event.ContentBlock.Name,
						Arguments: event.Delta.PartialJSON,
						Source:    "anthropic",
						Raw:       mustMarshalRaw(event),
					},
				}
			}
		}
	case "message_stop":
		result.Done = true
	}

	return result
}
