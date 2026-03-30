package react

import (
	"context"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/providers"
)

// ChatStream 发送消息（流式）
func (a *ReActAgent) ChatStream(ctx context.Context, msg bus.InboundMessage, callback StreamCallback) (string, int, error) {
	prepared, err := a.prepareChat(ctx, msg)
	if err != nil {
		if callback != nil {
			callback(StreamChunk{Error: err})
		}
		return "", 0, err
	}

	content, iteration, totalTokens, assistantMessageID, err := a.RunLLMStream(ctx, prepared.sessionKey, prepared.modelName, prepared.provider, prepared.messages, msg, callback)
	if err != nil {
		return "", iteration, err
	}
	_ = totalTokens

	a.finishAgentRun(ctx, msg, content, iteration, assistantMessageID, false)

	return content, iteration, nil
}

// RunLLMStream 运行LLM模型（流式）
func (a *ReActAgent) RunLLMStream(
	ctx context.Context,
	sessionKey string,
	modelName string,
	provider providers.Provider,
	messages []providers.ChatMessage,
	msg bus.InboundMessage,
	callback StreamCallback,
) (string, int, int, string, error) {
	state, content, err := a.runIterationLoop(ctx, msg, modelName, messages, func(start *iterationStart, state *llmLoopState) (string, bool, error) {
		collector, err := a.collectStreamIteration(ctx, provider, start.request, state.iteration, callback)
		if err != nil {
			return "", false, err
		}

		outcome := a.resolveStreamCollectorOutcome(collector, state.totalTokens)
		resolution, err := a.applyStreamIterationResolution(ctx, msg, sessionKey, state, outcome, callback)
		if err != nil {
			return "", false, err
		}

		if resolution.continueLoop {
			return "", false, nil
		}

		return collector.content(), true, nil
	}, func(state *llmLoopState) error {
		return a.failStreamRun(callback, state.iteration, a.maxToolIterationsError())
	})
	if err != nil {
		return "", state.iteration, state.totalTokens, state.assistantMessageID, err
	}

	return content, state.iteration, state.totalTokens, state.assistantMessageID, nil
}

// ChunkCallback 处理流式内容块
func (a *ReActAgent) ChunkCallback(
	filter *thinkStreamFilter,
	iteration int,
	collector *streamIterationCollector,
	callback StreamCallback,
) providers.StreamCallback {
	return func(chunk, reasoning string, toolCalls []providers.ToolCall, done bool) error {
		// chunk 和 reasoning 可能交错到达，filter 负责把隐藏思维和可见文本重新拼回统一事件流。
		extraction := filter.Push(chunk)
		reasoning = joinReasoningParts(extraction.Reasoning, reasoning)

		streamChunk := StreamChunk{
			Iteration: iteration,
			Content:   extraction.Visible,
			Reasoning: reasoning,
		}
		if collector != nil {
			collector.appendVisible(extraction.Visible)
			collector.appendReasoning(a, streamChunk.Reasoning, iteration)
			collector.appendToolCalls(toolCalls)
		}

		if callback != nil && (streamChunk.Content != "" || streamChunk.Reasoning != "") {
			if err := callback(streamChunk); err != nil {
				return err
			}
		}

		return nil
	}
}
