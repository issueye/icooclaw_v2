package react

import (
	"context"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/providers"
)

func (a *ReActAgent) runLLMHooksBefore(ctx context.Context, msg bus.InboundMessage, messages []providers.ChatMessage) ([]providers.ChatMessage, error) {
	if a.hooks == nil {
		return messages, nil
	}
	return a.hooks.OnRunLLMBefore(ctx, msg, messages)
}

func (a *ReActAgent) runLLMHooksAfter(ctx context.Context, msg bus.InboundMessage, messages []providers.ChatMessage) ([]providers.ChatMessage, error) {
	if a.hooks == nil {
		return messages, nil
	}
	return a.hooks.OnRunLLMAfter(ctx, msg, messages)
}

func (a *ReActAgent) finishAgentRun(
	ctx context.Context,
	msg bus.InboundMessage,
	content string,
	iteration int,
	assistantMessageID string,
	async bool,
) {
	if a.hooks == nil {
		return
	}

	run := func() {
		if err := a.hooks.OnAgentEnd(ctx, msg, content, iteration, assistantMessageID); err != nil {
			a.log().With("name", "【智能体】").Warn("执行 OnAgentEnd 钩子失败", "error", err)
		}
	}

	if async {
		go run()
		return
	}

	run()
}
