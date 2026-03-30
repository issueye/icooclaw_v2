package react

import (
	"testing"

	"icooclaw/pkg/providers"
)

func TestIsStreamIndexID(t *testing.T) {
	tests := []struct {
		id       string
		expected bool
	}{
		{"stream_index:0", true},
		{"stream_index:1", true},
		{"stream_index:10", true},
		{"call_abc123", false},
		{"", false},
		{"short", false},
		{"regular_id", false},
	}

	for _, tt := range tests {
		result := isStreamIndexID(tt.id)
		if result != tt.expected {
			t.Errorf("isStreamIndexID(%q) = %v, expected %v", tt.id, result, tt.expected)
		}
	}
}

func TestMergeToolCalls(t *testing.T) {
	agent := newTestAgentWithLogger()

	result := agent.mergeToolCalls(nil)
	if result != nil {
		t.Error("expected nil for empty input")
	}

	toolCalls := []providers.ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "test_tool",
				Arguments: `{"arg": "value"}`,
			},
		},
	}

	result = agent.mergeToolCalls(toolCalls)
	if len(result) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(result))
	}
	if result[0].Function.Name != "test_tool" {
		t.Errorf("expected 'test_tool', got '%s'", result[0].Function.Name)
	}
}

func TestMergeToolCalls_StreamIndex(t *testing.T) {
	agent := newTestAgentWithLogger()

	toolCalls := []providers.ToolCall{
		{
			ID:   "stream_index:0",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "test",
				Arguments: `{"a":`,
			},
		},
		{
			ID:   "stream_index:0",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Arguments: ` "1"}`,
			},
		},
	}

	result := agent.mergeToolCalls(toolCalls)
	if len(result) != 1 {
		t.Errorf("expected 1 merged tool call, got %d", len(result))
	}
	if result[0].Function.Name != "test" {
		t.Errorf("expected 'test', got '%s'", result[0].Function.Name)
	}
	if result[0].Function.Arguments != `{"a": "1"}` {
		t.Errorf("expected merged arguments, got '%s'", result[0].Function.Arguments)
	}
}

func TestValidateToolCalls(t *testing.T) {
	agent := newTestAgentWithLogger()

	result := agent.validateToolCalls(nil)
	if len(result) != 0 {
		t.Error("expected empty slice for nil input")
	}

	toolCalls := []providers.ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "valid_tool",
				Arguments: `{"arg": "value"}`,
			},
		},
		{
			ID:   "call_2",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "",
				Arguments: `{}`,
			},
		},
		{
			ID:   "call_3",
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      "no_args_tool",
				Arguments: "",
			},
		},
	}

	result = agent.validateToolCalls(toolCalls)
	if len(result) != 2 {
		t.Errorf("expected 2 valid tool calls, got %d", len(result))
	}

	for _, tc := range result {
		if tc.Function.Name == "no_args_tool" && tc.Function.Arguments != "{}" {
			t.Errorf("expected empty args to be '{}', got '%s'", tc.Function.Arguments)
		}
	}
}
