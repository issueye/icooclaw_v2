package adapter

import (
	"encoding/json"
	"testing"

	"icooclaw/pkg/providers/models/anthropic_model"
	"icooclaw/pkg/providers/models/openai_model"
)

func TestFromOpenAIResponse(t *testing.T) {
	resp := openai_model.Response{
		ID:    "resp-1",
		Model: "gpt-4o",
		Choices: []openai_model.Choice{
			{
				Message: openai_model.Message{
					Content:   "最终答案",
					Reasoning: "推理过程",
					ToolCalls: []openai_model.Tool{
						{
							ID:   "call_1",
							Type: "function",
							Function: openai_model.Function{
								Name:      "weather",
								Arguments: `{"city":"成都"}`,
							},
						},
					},
				},
			},
		},
		Usage: openai_model.Usage{PromptTokens: 10, CompletionTokens: 8, TotalTokens: 18},
	}

	got := FromOpenAIResponse(resp)
	if got.Content != "最终答案" {
		t.Fatalf("content = %q, want %q", got.Content, "最终答案")
	}
	if got.Reasoning != "推理过程" {
		t.Fatalf("reasoning = %q, want %q", got.Reasoning, "推理过程")
	}
	if got.Source != "openai" || len(got.Raw) == 0 {
		t.Fatalf("expected openai raw payload, got source=%q raw=%q", got.Source, string(got.Raw))
	}
	if len(got.ToolCalls) != 1 || got.ToolCalls[0].Name != "weather" {
		t.Fatalf("tool calls = %+v, want weather tool", got.ToolCalls)
	}
	if got.ToolCalls[0].Source != "openai" || len(got.ToolCalls[0].Raw) == 0 {
		t.Fatalf("expected raw tool call payload, got %+v", got.ToolCalls[0])
	}
}

func TestFromAnthropicResponse(t *testing.T) {
	input, _ := json.Marshal(map[string]any{"city": "成都"})
	resp := anthropic_model.Response{
		ID:    "msg_1",
		Model: "claude-3-5-sonnet",
		Content: []anthropic_model.ContentBlock{
			{Type: "thinking", Thinking: "先分析"},
			{Type: "text", Text: "最终答案"},
			{Type: "tool_use", ID: "toolu_1", Name: "weather", Input: input},
		},
		Usage: anthropic_model.Usage{InputTokens: 12, OutputTokens: 15},
	}

	got := FromAnthropicResponse(resp)
	if got.Content != "最终答案" {
		t.Fatalf("content = %q, want %q", got.Content, "最终答案")
	}
	if got.Reasoning != "先分析" {
		t.Fatalf("reasoning = %q, want %q", got.Reasoning, "先分析")
	}
	if got.Source != "anthropic" || len(got.Raw) == 0 {
		t.Fatalf("expected anthropic raw payload, got source=%q raw=%q", got.Source, string(got.Raw))
	}
	if len(got.ToolCalls) != 1 || got.ToolCalls[0].Name != "weather" {
		t.Fatalf("tool calls = %+v, want weather tool", got.ToolCalls)
	}
	if got.ToolCalls[0].Source != "anthropic" || len(got.ToolCalls[0].Raw) == 0 {
		t.Fatalf("expected raw tool call payload, got %+v", got.ToolCalls[0])
	}
}

func TestFromAnthropicEvent(t *testing.T) {
	event := anthropic_model.Event{
		Type:  "content_block_delta",
		Index: 0,
		Delta: anthropic_model.Delta{
			Type:        "input_json_delta",
			PartialJSON: `{"city":"成`,
		},
		ContentBlock: anthropic_model.ContentBlock{
			Type: "tool_use",
			ID:   "toolu_1",
			Name: "weather",
		},
	}

	got := FromAnthropicEvent(event)
	if len(got.ToolCalls) != 1 {
		t.Fatalf("tool call count = %d, want 1", len(got.ToolCalls))
	}
	if got.ToolCalls[0].Arguments != `{"city":"成` {
		t.Fatalf("tool arguments = %q, want partial json", got.ToolCalls[0].Arguments)
	}
	if got.Source != "anthropic" || len(got.Raw) == 0 {
		t.Fatalf("expected raw event payload, got source=%q raw=%q", got.Source, string(got.Raw))
	}
}

func TestToAnthropicRequest_NormalizesRequiredSchema(t *testing.T) {
	req := Request{
		Model: "claude-3-5-sonnet",
		Tools: []Tool{
			{
				Name:        "weather",
				Description: "查询天气",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"destination": map[string]any{
							"type":        "string",
							"description": "目标城市",
							"required":    true,
						},
						"unit": map[string]any{
							"type":        "string",
							"description": "温度单位",
						},
					},
				},
			},
		},
	}

	got := ToAnthropicRequest(req)
	if len(got.Tools) != 1 {
		t.Fatalf("tool count = %d, want 1", len(got.Tools))
	}

	properties, ok := got.Tools[0].InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties type = %T, want map[string]any", got.Tools[0].InputSchema["properties"])
	}
	destination, ok := properties["destination"].(map[string]any)
	if !ok {
		t.Fatalf("destination schema type = %T, want map[string]any", properties["destination"])
	}
	if _, exists := destination["required"]; exists {
		t.Fatalf("expected destination.required to be removed, got %+v", destination["required"])
	}

	required, ok := got.Tools[0].InputSchema["required"].([]any)
	if !ok {
		t.Fatalf("required type = %T, want []any", got.Tools[0].InputSchema["required"])
	}
	if len(required) != 1 || required[0] != "destination" {
		t.Fatalf("required = %#v, want [\"destination\"]", required)
	}
}

func TestToAnthropicRequest_ConvertsToolMessagesToAnthropicBlocks(t *testing.T) {
	req := Request{
		Model: "claude-3-5-sonnet",
		Messages: []Message{
			{
				Role:    "user",
				Content: "查一下天气",
			},
			{
				Role:    "assistant",
				Content: "我来调用工具。",
				ToolCalls: []ToolCall{
					{
						ID:        "call_1",
						Name:      "weather",
						Arguments: `{"city":"成都"}`,
					},
				},
			},
			{
				Role:       "tool",
				Content:    "晴，24C",
				ToolCallID: "call_1",
			},
		},
	}

	got := ToAnthropicRequest(req)
	if len(got.Messages) != 3 {
		t.Fatalf("message count = %d, want 3", len(got.Messages))
	}

	assistantBlocks, ok := got.Messages[1].Content.([]anthropic_model.ContentBlock)
	if !ok {
		t.Fatalf("assistant content type = %T, want []anthropic_model.ContentBlock", got.Messages[1].Content)
	}
	if len(assistantBlocks) != 2 || assistantBlocks[1].Type != "tool_use" {
		t.Fatalf("assistant blocks = %#v, want text + tool_use", assistantBlocks)
	}
	if assistantBlocks[1].ID != "call_1" || assistantBlocks[1].Name != "weather" {
		t.Fatalf("assistant tool block = %#v, want call_1/weather", assistantBlocks[1])
	}

	toolBlocks, ok := got.Messages[2].Content.([]anthropic_model.ContentBlock)
	if !ok {
		t.Fatalf("tool result content type = %T, want []anthropic_model.ContentBlock", got.Messages[2].Content)
	}
	if got.Messages[2].Role != "user" {
		t.Fatalf("tool result role = %q, want user", got.Messages[2].Role)
	}
	if len(toolBlocks) != 1 || toolBlocks[0].Type != "tool_result" {
		t.Fatalf("tool blocks = %#v, want one tool_result block", toolBlocks)
	}
	if toolBlocks[0].ToolUseID != "call_1" || toolBlocks[0].Content != "晴，24C" {
		t.Fatalf("tool result block = %#v, want tool_use_id call_1 and content", toolBlocks[0])
	}
}

func TestFromAnthropicEvent_ContentBlockStartToolUseEmitsToolCall(t *testing.T) {
	event := anthropic_model.Event{
		Type: "content_block_start",
		ContentBlock: anthropic_model.ContentBlock{
			Type:  "tool_use",
			ID:    "call_1",
			Name:  "weather",
			Input: json.RawMessage(`{"city":"成都"}`),
		},
	}

	got := FromAnthropicEvent(event)
	if len(got.ToolCalls) != 1 {
		t.Fatalf("tool call count = %d, want 1", len(got.ToolCalls))
	}
	if got.ToolCalls[0].Name != "weather" || got.ToolCalls[0].Arguments != `{"city":"成都"}` {
		t.Fatalf("tool call = %+v, want weather/{\"city\":\"成都\"}", got.ToolCalls[0])
	}
}
