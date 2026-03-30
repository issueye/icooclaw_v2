package openai_responses_model

import (
	"encoding/json"
	"testing"
)

func TestRequest_Marshal(t *testing.T) {
	store := false
	parallel := true
	req := Request{
		Model: "gpt-5",
		Input: []InputItem{
			{
				Type: "message",
				Role: "user",
				Content: []ContentItem{
					{Type: "input_text", Text: "你好"},
				},
			},
		},
		Instructions:      "You are a helpful assistant.",
		Store:             &store,
		ParallelToolCalls: &parallel,
		Tools: []Tool{
			{
				Type: "function",
				Function: &FunctionTool{
					Name:        "weather",
					Description: "获取天气",
					Parameters: map[string]any{
						"type": "object",
					},
				},
			},
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

	if decoded["model"] != "gpt-5" {
		t.Fatalf("model = %v, want %q", decoded["model"], "gpt-5")
	}
	if _, ok := decoded["tools"]; !ok {
		t.Fatal("expected tools field")
	}
	if _, ok := decoded["input"]; !ok {
		t.Fatal("expected input field")
	}
}

func TestResponseAndStreamEvent_Unmarshal(t *testing.T) {
	responsePayload := []byte(`{
		"id":"resp_123",
		"object":"response",
		"created_at":1740000000,
		"status":"completed",
		"model":"gpt-5",
		"output":[
			{"id":"msg_1","type":"message","status":"completed","role":"assistant","content":[{"type":"output_text","text":"你好"}]},
			{"id":"fc_1","type":"function_call","status":"completed","call_id":"call_1","name":"weather","arguments":"{\"city\":\"成都\"}"}
		],
		"usage":{"input_tokens":10,"output_tokens":12,"total_tokens":22}
	}`)

	var resp Response
	if err := json.Unmarshal(responsePayload, &resp); err != nil {
		t.Fatalf("Unmarshal response error = %v", err)
	}

	if resp.ID != "resp_123" {
		t.Fatalf("response id = %q, want %q", resp.ID, "resp_123")
	}
	if len(resp.Output) != 2 {
		t.Fatalf("output count = %d, want 2", len(resp.Output))
	}
	if resp.Output[1].CallID != "call_1" {
		t.Fatalf("call_id = %q, want %q", resp.Output[1].CallID, "call_1")
	}

	eventPayload := []byte(`{
		"type":"response.output_text.delta",
		"sequence_number":2,
		"response_id":"resp_123",
		"output_index":0,
		"item_id":"msg_1",
		"delta":"你好"
	}`)

	var event StreamEvent
	if err := json.Unmarshal(eventPayload, &event); err != nil {
		t.Fatalf("Unmarshal event error = %v", err)
	}

	if event.Type != "response.output_text.delta" {
		t.Fatalf("event type = %q, want %q", event.Type, "response.output_text.delta")
	}
	if event.Delta != "你好" {
		t.Fatalf("event delta = %q, want %q", event.Delta, "你好")
	}
}
