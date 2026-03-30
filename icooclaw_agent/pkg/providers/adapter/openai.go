package adapter

import (
	"fmt"

	"icooclaw/pkg/providers/models/openai_model"
)

// ToOpenAIRequest 将中间层请求转换为 OpenAI 兼容请求。
func ToOpenAIRequest(req Request) openai_model.Request {
	result := openai_model.Request{
		Model:       req.Model,
		Stream:      req.Stream,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Messages:    make([]openai_model.ChatMessage, 0, len(req.Messages)),
		Tools:       make([]openai_model.RequestTool, 0, len(req.Tools)),
	}

	for _, message := range req.Messages {
		result.Messages = append(result.Messages, openai_model.ChatMessage{
			Role:       message.Role,
			Name:       message.Name,
			Content:    message.Content,
			ToolCallID: message.ToolCallID,
			ToolCalls:  toOpenAIToolCalls(message.ToolCalls),
		})
	}

	for _, tool := range req.Tools {
		result.Tools = append(result.Tools, openai_model.RequestTool{
			Type: tool.Type,
			Function: openai_model.Function{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}

	return result
}

// FromOpenAIResponse 将 OpenAI 兼容响应转换为中间层响应。
func FromOpenAIResponse(resp openai_model.Response) Response {
	result := Response{
		ID:      resp.ID,
		Model:   resp.Model,
		Object:  resp.Object,
		Created: resp.Created,
		Source:  "openai",
		Raw:     mustMarshalRaw(resp),
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	if len(resp.Choices) == 0 {
		return result
	}

	result.Content = resp.Choices[0].Message.Content
	result.Reasoning = joinReasoning(resp.Choices[0].Message.Reasoning, resp.Choices[0].Message.ReasoningDetails)
	result.ToolCalls = fromOpenAIToolCalls(resp.Choices[0].Message.ToolCalls)
	return result
}

// FromOpenAIStreamResponse 将 OpenAI 兼容流式 chunk 转换为中间层事件。
func FromOpenAIStreamResponse(resp openai_model.Response) StreamEvent {
	event := StreamEvent{
		Source: "openai",
		Raw:    mustMarshalRaw(resp),
	}
	if len(resp.Choices) == 0 {
		return event
	}

	choice := resp.Choices[0]
	event.Content = choice.Delta.Content
	event.Reasoning = joinReasoning(choice.Delta.Reasoning, choice.Message.ReasoningDetails)
	event.ToolCalls = fromOpenAIToolCalls(choice.Delta.ToolCalls)
	event.Done = choice.FinishReason != ""
	return event
}

func toOpenAIToolCalls(toolCalls []ToolCall) []openai_model.Tool {
	if len(toolCalls) == 0 {
		return nil
	}

	result := make([]openai_model.Tool, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		result = append(result, openai_model.Tool{
			ID:   toolCall.ID,
			Type: toolCall.Type,
			Function: openai_model.Function{
				Name:      toolCall.Name,
				Arguments: toolCall.Arguments,
			},
		})
	}
	return result
}

func fromOpenAIToolCalls(toolCalls []openai_model.Tool) []ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	result := make([]ToolCall, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		id := toolCall.ID
		if id == "" && toolCall.Index >= 0 {
			id = fmt.Sprintf("stream_index:%d", toolCall.Index)
		}
		result = append(result, ToolCall{
			ID:        id,
			Type:      toolCall.Type,
			Name:      toolCall.Function.Name,
			Arguments: toolCall.Function.Arguments,
			Source:    "openai",
			Raw:       mustMarshalRaw(toolCall),
		})
	}
	return result
}

func joinReasoning(reasoning string, details []openai_model.ReasoningDetail) string {
	if reasoning != "" {
		return reasoning
	}
	if len(details) == 0 {
		return ""
	}

	var joined string
	for _, detail := range details {
		if detail.Text == "" {
			continue
		}
		if joined != "" {
			joined += "\n"
		}
		joined += detail.Text
	}
	return joined
}
