package adapter

import (
	"encoding/json"

	"icooclaw/pkg/providers/protocol"
)

func ToAdapterRequest(req protocol.ChatRequest) Request {
	result := Request{
		Model:       req.Model,
		Messages:    make([]Message, 0, len(req.Messages)),
		Tools:       make([]Tool, 0, len(req.Tools)),
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream,
		Source:      "providers",
	}
	if data, err := json.Marshal(req); err == nil {
		result.Raw = data
	}

	for _, message := range req.Messages {
		var raw json.RawMessage
		if data, err := json.Marshal(message); err == nil {
			raw = data
		}
		result.Messages = append(result.Messages, Message{
			Role:       message.Role,
			Name:       message.Name,
			Content:    message.Content,
			ToolCallID: message.ToolCallID,
			ToolCalls:  ToAdapterToolCalls(message.ToolCalls),
			Source:     "providers",
			Raw:        raw,
		})
	}

	for _, tool := range req.Tools {
		result.Tools = append(result.Tools, Tool{
			Type:        tool.Type,
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			Parameters:  tool.Function.Parameters,
		})
	}

	return result
}

func FromAdapterResponse(resp Response) *protocol.ChatResponse {
	return &protocol.ChatResponse{
		ID:        resp.ID,
		Model:     resp.Model,
		Object:    resp.Object,
		Created:   resp.Created,
		Content:   resp.Content,
		Reasoning: resp.Reasoning,
		ToolCalls: fromAdapterToolCalls(resp.ToolCalls),
		Usage: protocol.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
}

func FromAdapterStreamEvent(event StreamEvent) (content string, reasoning string, toolCalls []protocol.ToolCall, done bool) {
	return event.Content, event.Reasoning, fromAdapterToolCalls(event.ToolCalls), event.Done
}

func ToAdapterToolCalls(toolCalls []protocol.ToolCall) []ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	result := make([]ToolCall, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		var raw json.RawMessage
		if data, err := json.Marshal(toolCall); err == nil {
			raw = data
		}
		result = append(result, ToolCall{
			ID:        toolCall.ID,
			Type:      toolCall.Type,
			Name:      toolCall.Function.Name,
			Arguments: toolCall.Function.Arguments,
			Source:    "providers",
			Raw:       raw,
		})
	}
	return result
}

func fromAdapterToolCalls(toolCalls []ToolCall) []protocol.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	result := make([]protocol.ToolCall, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		result = append(result, protocol.ToolCall{
			ID:   toolCall.ID,
			Type: toolCall.Type,
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      toolCall.Name,
				Arguments: toolCall.Arguments,
			},
		})
	}
	return result
}
