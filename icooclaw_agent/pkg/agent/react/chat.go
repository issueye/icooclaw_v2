package react

import (
	"context"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/providers"
)

// Chat 发送消息（非流式）
func (a *ReActAgent) Chat(ctx context.Context, msg bus.InboundMessage) (string, int, error) {
	prepared, err := a.prepareChat(ctx, msg)
	if err != nil {
		return "", 0, err
	}

	content, iteration, traceItems, totalTokens, err := a.RunLLM(ctx, prepared.modelName, prepared.provider, prepared.messages, msg)
	if err != nil {
		return "", 0, err
	}

	var assistantMessageID string
	if content != "" {
		assistantMessageID, err = a.saveAssistantMessage(ctx, prepared.sessionKey, content, traceItems, iteration, totalTokens)
		if err != nil {
			a.log().With("name", "【智能体】").Warn("保存助手消息失败", "error", err)
		}
	}

	// 非流式路径保持异步收尾，避免额外阻塞主流程。
	a.finishAgentRun(ctx, msg, content, iteration, assistantMessageID, true)

	return content, iteration, nil
}

// RunLLM 运行LLM模型（非流式）
func (a *ReActAgent) RunLLM(
	ctx context.Context,
	modelName string,
	provider providers.Provider,
	messages []providers.ChatMessage,
	msg bus.InboundMessage,
) (string, int, []MessageTraceItem, int, error) {
	state, content, err := a.runIterationLoop(ctx, msg, modelName, messages, func(start *iterationStart, state *llmLoopState) (string, bool, error) {
		outcome, err := a.collectSyncIteration(ctx, provider, start.request, state.iteration, state.traceItems)
		if err != nil {
			return "", false, err
		}
		state.totalTokens = outcome.totalTokens

		resolution, err := a.resolveIterationOutcome(ctx, msg, state.currentMessages, *outcome, state.iteration, nil)
		if err != nil {
			return "", false, err
		}
		state.currentMessages = resolution.messages
		state.traceItems = resolution.traceItems

		if resolution.continueLoop {
			return "", false, nil
		}

		return outcome.content, true, nil
	}, nil)
	if err != nil {
		return "", state.iteration, state.traceItems, state.totalTokens, err
	}

	return content, state.iteration, state.traceItems, state.totalTokens, nil
}
