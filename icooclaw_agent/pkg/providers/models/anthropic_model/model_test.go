package anthropic_model

import (
	"encoding/json"
	"testing"
)

func TestRequest_Marshal(t *testing.T) {
	req := Request{
		Model:     "claude-3-5-sonnet-20241022",
		MaxTokens: 4096,
		System: []ContentBlock{
			{Type: "text", Text: "You are a helpful assistant."},
		},
		Messages: []Message{
			{
				Role: "user",
				Content: []ContentBlock{
					{Type: "text", Text: "帮我查天气"},
				},
			},
		},
		Tools: []Tool{
			{
				Name:        "weather",
				Description: "获取天气",
				InputSchema: map[string]any{
					"type": "object",
				},
			},
		},
		Stream: true,
		Thinking: &ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: 1024,
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded["model"] != "claude-3-5-sonnet-20241022" {
		t.Fatalf("model = %v, want %q", decoded["model"], "claude-3-5-sonnet-20241022")
	}
	if _, ok := decoded["messages"]; !ok {
		t.Fatal("expected messages field")
	}
	if _, ok := decoded["tools"]; !ok {
		t.Fatal("expected tools field")
	}
}

func TestResponseAndEvent_Unmarshal(t *testing.T) {
	responsePayload := []byte(`{
		"id":"msg_123",
		"type":"message",
		"role":"assistant",
		"model":"claude-3-5-sonnet-20241022",
		"content":[
			{"type":"thinking","thinking":"先分析"},
			{"type":"text","text":"最终答案"},
			{"type":"tool_use","id":"toolu_1","name":"weather","input":{"city":"成都"}}
		],
		"stop_reason":"tool_use",
		"usage":{"input_tokens":12,"output_tokens":18}
	}`)

	var resp Response
	if err := json.Unmarshal(responsePayload, &resp); err != nil {
		t.Fatalf("Unmarshal response error = %v", err)
	}

	if resp.ID != "msg_123" {
		t.Fatalf("response id = %q, want %q", resp.ID, "msg_123")
	}
	if len(resp.Content) != 3 {
		t.Fatalf("content count = %d, want 3", len(resp.Content))
	}
	if resp.Content[2].Name != "weather" {
		t.Fatalf("tool name = %q, want %q", resp.Content[2].Name, "weather")
	}

	eventPayload := []byte(`{
		"type":"content_block_delta",
		"index":1,
		"delta":{"type":"text_delta","text":"答案"}
	}`)

	var event Event
	if err := json.Unmarshal(eventPayload, &event); err != nil {
		t.Fatalf("Unmarshal event error = %v", err)
	}

	if event.Type != "content_block_delta" {
		t.Fatalf("event type = %q, want %q", event.Type, "content_block_delta")
	}
	if event.Delta.Text != "答案" {
		t.Fatalf("event text = %q, want %q", event.Delta.Text, "答案")
	}
}
