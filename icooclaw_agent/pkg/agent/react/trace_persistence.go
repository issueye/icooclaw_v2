package react

import (
	"context"
	"encoding/json"
	"strings"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/storage"
)

func (a *ReActAgent) appendThinkingTrace(items []MessageTraceItem, content string, iteration int) []MessageTraceItem {
	extraction := extractThinkBlocks(content)
	if extraction.Reasoning != "" {
		content = extraction.Reasoning
	} else {
		content = strings.TrimSpace(content)
	}
	if content == "" {
		return items
	}

	if len(items) > 0 && items[len(items)-1].Type == "thinking" {
		items[len(items)-1].Content += content
		return items
	}

	return append(items, MessageTraceItem{
		Type:      "thinking",
		Content:   content,
		Iteration: iteration,
	})
}

func (a *ReActAgent) appendToolTrace(items []MessageTraceItem, tc providers.ToolCall, iteration int) []MessageTraceItem {
	return append(items, MessageTraceItem{
		Type:       "tool_call",
		ToolCallID: tc.ID,
		ToolName:   tc.Function.Name,
		ToolArgs:   tc.Function.Arguments,
		Iteration:  iteration,
	})
}

func (a *ReActAgent) attachToolResult(items []MessageTraceItem, toolCallID, result string, iteration int) []MessageTraceItem {
	result = extractThinkBlocks(result).Visible
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].Type == "tool_call" && items[i].ToolCallID == toolCallID {
			items[i].ToolResult = result
			if items[i].Iteration == 0 {
				items[i].Iteration = iteration
			}
			return items
		}
	}

	return append(items, MessageTraceItem{
		Type:       "tool_call",
		ToolCallID: toolCallID,
		ToolResult: result,
		Iteration:  iteration,
	})
}

func buildAssistantMessageMetadata(traceItems []MessageTraceItem, iteration int) string {
	if len(traceItems) == 0 && iteration == 0 {
		return ""
	}

	meta := AssistantMessageMetadata{
		Type:       "assistant_trace",
		Iteration:  iteration,
		TraceItems: traceItems,
	}
	for _, item := range traceItems {
		if item.Type == "thinking" {
			meta.ReasoningContent += item.Content
		}
	}

	data, _ := json.Marshal(meta)
	return string(data)
}

func extractThinkingContent(traceItems []MessageTraceItem) string {
	if len(traceItems) == 0 {
		return ""
	}

	parts := make([]string, 0, len(traceItems))
	for _, item := range traceItems {
		if item.Type != "thinking" {
			continue
		}
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		parts = append(parts, content)
	}

	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func sanitizeTraceItems(traceItems []MessageTraceItem) []MessageTraceItem {
	if len(traceItems) == 0 {
		return nil
	}

	items := make([]MessageTraceItem, 0, len(traceItems))
	for _, item := range traceItems {
		if item.Type == "thinking" {
			extraction := extractThinkBlocks(item.Content)
			if extraction.Reasoning != "" {
				item.Content = extraction.Reasoning
			} else {
				item.Content = strings.TrimSpace(item.Content)
			}
		}
		if item.ToolResult != "" {
			item.ToolResult = extractThinkBlocks(item.ToolResult).Visible
		}
		items = append(items, item)
	}
	return items
}

func (a *ReActAgent) saveAssistantMessage(ctx context.Context, sessionKey, content string, traceItems []MessageTraceItem, iteration int, totalTokens int) (string, error) {
	_ = ctx
	if a.storage == nil {
		return "", nil
	}
	content = stripThinkBlocks(content)
	traceItems = sanitizeTraceItems(traceItems)
	if content == "" && len(traceItems) == 0 {
		return "", nil
	}

	msg := &storage.Message{
		SessionID:   sessionKey,
		Role:        consts.RoleAssistant,
		Content:     content,
		Thinking:    extractThinkingContent(traceItems),
		TotalTokens: totalTokens,
		Metadata:    buildAssistantMessageMetadata(traceItems, iteration),
	}
	if err := a.storage.Message().Save(msg); err != nil {
		return "", err
	}
	return msg.ID, nil
}
