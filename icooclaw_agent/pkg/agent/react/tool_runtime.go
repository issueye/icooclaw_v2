package react

import (
	"context"
	"fmt"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
)

func (a *ReActAgent) executeToolCalls(
	ctx context.Context,
	msg bus.InboundMessage,
	toolCalls []providers.ToolCall,
	iteration int,
	callback StreamCallback,
) ([]providers.ChatMessage, []MessageTraceItem, error) {
	currentMessages := make([]providers.ChatMessage, 0, len(toolCalls))
	traceItems := make([]MessageTraceItem, 0, len(toolCalls))

	for _, tc := range toolCalls {
		traceItems = a.appendToolTrace(traceItems, tc, iteration)

		if callback != nil {
			if err := callback(StreamChunk{
				ToolCallID: tc.ID,
				ToolName:   tc.Function.Name,
				ToolArgs:   tc.Function.Arguments,
				Iteration:  iteration,
			}); err != nil {
				return nil, traceItems, err
			}
		}

		toolResult, execErr := a.executeToolCall(ctx, tc, msg)
		if execErr != nil {
			toolResult = fmt.Sprintf("错误: %v", execErr)
		}

		toolResultExtraction := extractThinkBlocks(toolResult)
		traceItems = a.appendThinkingTrace(traceItems, toolResultExtraction.Reasoning, iteration)
		toolResult = toolResultExtraction.Visible
		traceItems = a.attachToolResult(traceItems, tc.ID, toolResult, iteration)

		if callback != nil && toolResultExtraction.Reasoning != "" {
			if err := callback(StreamChunk{
				ToolCallID: tc.ID,
				ToolName:   tc.Function.Name,
				Reasoning:  toolResultExtraction.Reasoning,
				Iteration:  iteration,
			}); err != nil {
				return nil, traceItems, err
			}
		}

		if callback != nil {
			if err := callback(StreamChunk{
				ToolCallID: tc.ID,
				ToolName:   tc.Function.Name,
				ToolResult: toolResult,
				Iteration:  iteration,
			}); err != nil {
				return nil, traceItems, err
			}
		}

		currentMessages = append(currentMessages, providers.ChatMessage{
			Role:       consts.RoleTool.ToString(),
			Content:    toolResult,
			ToolCallID: tc.ID,
		})
	}

	return currentMessages, traceItems, nil
}

func (a *ReActAgent) appendAssistantToolMessages(
	currentMessages []providers.ChatMessage,
	content string,
	toolCalls []providers.ToolCall,
	totalTokens int,
	toolMessages []providers.ChatMessage,
) []providers.ChatMessage {
	assistantMsg := providers.ChatMessage{
		Role:        consts.RoleAssistant.ToString(),
		Content:     content,
		ToolCalls:   toolCalls,
		TotalTokens: totalTokens,
	}
	currentMessages = append(currentMessages, assistantMsg)

	for i := range toolMessages {
		toolMessages[i].TotalTokens = totalTokens
	}

	return append(currentMessages, toolMessages...)
}
