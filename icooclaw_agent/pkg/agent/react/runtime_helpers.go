package react

import (
	"context"
	"encoding/json"
	"fmt"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/providers"
)

func (a *ReActAgent) buildChatRequest(modelName string, messages []providers.ChatMessage) providers.ChatRequest {
	req := providers.ChatRequest{
		Model:    modelName,
		Messages: messages,
	}

	toolDefs := a.tools.ToProviderDefs()
	if len(toolDefs) > 0 {
		req.Tools = a.convertToolDefinitions(toolDefs)
	}

	return req
}

func (a *ReActAgent) beginLLMIteration(
	ctx context.Context,
	msg bus.InboundMessage,
	modelName string,
	messages []providers.ChatMessage,
) (*iterationStart, error) {
	// 先跑 before hook，再基于 hook 修改后的消息构造请求，避免同步/流式语义漂移。
	currentMessages, err := a.runLLMHooksBefore(ctx, msg, messages)
	if err != nil {
		return nil, err
	}
	currentMessages = sanitizeProviderMessages(currentMessages)

	request := a.buildChatRequest(modelName, currentMessages)
	a.log().With("name", "【智能体】").Info("发送LLM prompt",
		"session_id", msg.SessionID,
		"model", modelName,
		"prompt", formatPromptForLog(request.Messages))

	return &iterationStart{
		messages: currentMessages,
		request:  request,
	}, nil
}

func formatPromptForLog(messages []providers.ChatMessage) string {
	data, err := json.Marshal(messages)
	if err != nil {
		return fmt.Sprintf("marshal prompt failed: %v", err)
	}
	return string(data)
}

func (a *ReActAgent) completeLLMIteration(
	ctx context.Context,
	msg bus.InboundMessage,
	currentMessages []providers.ChatMessage,
	content string,
	toolCalls []providers.ToolCall,
	totalTokens int,
	toolMessages []providers.ChatMessage,
) ([]providers.ChatMessage, error) {
	// 每轮结束统一补齐 assistant/tool message，再执行 after hook，保证两条路径的收尾顺序一致。
	currentMessages = a.appendAssistantToolMessages(currentMessages, content, toolCalls, totalTokens, toolMessages)
	return a.runLLMHooksAfter(ctx, msg, currentMessages)
}

func (a *ReActAgent) saveAssistantMessageSafe(
	ctx context.Context,
	sessionKey, content string,
	traceItems []MessageTraceItem,
	iteration int,
	totalTokens int,
) string {
	// 保存失败只记日志，由调用方继续返回主结果，避免把“可恢复的持久化失败”升级成主链路错误。
	assistantMessageID, err := a.saveAssistantMessage(ctx, sessionKey, content, traceItems, iteration, totalTokens)
	if err != nil {
		a.log().With("name", "【智能体】").Warn("保存助手消息失败", "error", err)
		return ""
	}

	return assistantMessageID
}

func (a *ReActAgent) sendStreamDone(callback StreamCallback, iteration int, totalTokens int) error {
	if callback == nil {
		return nil
	}

	return callback(StreamChunk{Done: true, Iteration: iteration, TotalTokens: totalTokens})
}

func (a *ReActAgent) resolveIterationOutcome(
	ctx context.Context,
	msg bus.InboundMessage,
	currentMessages []providers.ChatMessage,
	outcome iterationOutcome,
	iteration int,
	callback StreamCallback,
) (*iterationResolution, error) {
	if len(outcome.toolCalls) > 0 {
		toolMessages, toolTraceItems, err := a.executeToolCalls(ctx, msg, outcome.toolCalls, iteration, callback)
		if err != nil {
			return nil, err
		}
		outcome.traceItems = append(outcome.traceItems, toolTraceItems...)

		currentMessages, err = a.completeLLMIteration(
			ctx,
			msg,
			currentMessages,
			outcome.content,
			outcome.toolCalls,
			outcome.totalTokens,
			toolMessages,
		)
		if err != nil {
			return nil, err
		}

		return &iterationResolution{
			messages:     currentMessages,
			traceItems:   outcome.traceItems,
			continueLoop: true,
		}, nil
	}

	if err := a.sendStreamDone(callback, iteration, outcome.totalTokens); err != nil {
		return nil, err
	}

	currentMessages, err := a.completeLLMIteration(
		ctx,
		msg,
		currentMessages,
		outcome.content,
		nil,
		outcome.totalTokens,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &iterationResolution{
		messages:   currentMessages,
		traceItems: outcome.traceItems,
	}, nil
}

func (a *ReActAgent) resolveStreamIteration(
	ctx context.Context,
	msg bus.InboundMessage,
	currentMessages []providers.ChatMessage,
	sessionKey string,
	outcome iterationOutcome,
	iteration int,
	callback StreamCallback,
) (*iterationResolution, string, error) {
	resolution, err := a.resolveIterationOutcome(ctx, msg, currentMessages, outcome, iteration, callback)
	if err != nil {
		return nil, "", err
	}

	assistantMessageID := a.saveAssistantMessageSafe(
		ctx,
		sessionKey,
		outcome.content,
		resolution.traceItems,
		iteration,
		outcome.totalTokens,
	)

	return resolution, assistantMessageID, nil
}
