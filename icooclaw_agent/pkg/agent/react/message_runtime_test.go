package react

import (
	"testing"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
)

func TestSanitizeProviderMessages_DropsDirtyHistoryEntries(t *testing.T) {
	messages := []providers.ChatMessage{
		{Role: consts.RoleSystem.ToString(), Content: " system prompt "},
		{Role: consts.RoleUser.ToString(), Content: "旧问题"},
		{Role: consts.RoleAssistant.ToString(), Content: "   "},
		{Role: consts.RoleAssistant.ToString(), Content: "历史答案"},
		{Role: consts.RoleTool.ToString(), Content: "", ToolCallID: "call-orphan"},
	}

	sanitized := sanitizeProviderMessages(messages)
	if len(sanitized) != 3 {
		t.Fatalf("sanitized message count = %d, want %d", len(sanitized), 3)
	}
	if sanitized[0].Content != "system prompt" {
		t.Fatalf("sanitized system content = %q, want %q", sanitized[0].Content, "system prompt")
	}
	for _, message := range sanitized {
		if message.Role == consts.RoleAssistant.ToString() && message.Content == "" && len(message.ToolCalls) == 0 {
			t.Fatalf("unexpected empty assistant message kept: %+v", message)
		}
	}
}

func TestSanitizeProviderMessages_PreservesAssistantToolCalls(t *testing.T) {
	messages := []providers.ChatMessage{
		{
			Role:    consts.RoleAssistant.ToString(),
			Content: " ",
			ToolCalls: []providers.ToolCall{
				{
					ID:   " call_1 ",
					Type: "",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      " lookup ",
						Arguments: " ",
					},
				},
			},
		},
		{
			Role:       consts.RoleTool.ToString(),
			Content:    "工具结果",
			ToolCallID: " call_1 ",
		},
	}

	sanitized := sanitizeProviderMessages(messages)
	if len(sanitized) != 2 {
		t.Fatalf("sanitized message count = %d, want %d", len(sanitized), 2)
	}
	if len(sanitized[0].ToolCalls) != 1 {
		t.Fatalf("assistant tool call count = %d, want %d", len(sanitized[0].ToolCalls), 1)
	}
	if sanitized[0].ToolCalls[0].Type != "function" {
		t.Fatalf("tool call type = %q, want %q", sanitized[0].ToolCalls[0].Type, "function")
	}
	if sanitized[0].ToolCalls[0].Function.Name != "lookup" {
		t.Fatalf("tool call name = %q, want %q", sanitized[0].ToolCalls[0].Function.Name, "lookup")
	}
	if sanitized[0].ToolCalls[0].Function.Arguments != "{}" {
		t.Fatalf("tool call arguments = %q, want %q", sanitized[0].ToolCalls[0].Function.Arguments, "{}")
	}
	if sanitized[1].ToolCallID != "call_1" {
		t.Fatalf("tool message call id = %q, want %q", sanitized[1].ToolCallID, "call_1")
	}
}
