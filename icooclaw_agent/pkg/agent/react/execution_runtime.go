package react

import (
	"context"
	"fmt"
	"strings"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/providers"
)

type iterationStart struct {
	messages []providers.ChatMessage
	request  providers.ChatRequest
}

type iterationOutcome struct {
	content     string
	toolCalls   []providers.ToolCall
	totalTokens int
	traceItems  []MessageTraceItem
}

type iterationResolution struct {
	messages     []providers.ChatMessage
	traceItems   []MessageTraceItem
	continueLoop bool
}

type llmLoopState struct {
	currentMessages    []providers.ChatMessage
	traceItems         []MessageTraceItem
	iteration          int
	totalTokens        int
	assistantMessageID string
}

type streamIterationCollector struct {
	visibleContent strings.Builder
	toolCalls      []providers.ToolCall
	traceItems     []MessageTraceItem
}

func (c *streamIterationCollector) appendVisible(content string) {
	if content != "" {
		c.visibleContent.WriteString(content)
	}
}

func (c *streamIterationCollector) appendReasoning(agent *ReActAgent, reasoning string, iteration int) {
	if reasoning != "" {
		c.traceItems = agent.appendThinkingTrace(c.traceItems, reasoning, iteration)
	}
}

func (c *streamIterationCollector) appendToolCalls(toolCalls []providers.ToolCall) {
	if len(toolCalls) > 0 {
		c.toolCalls = append(c.toolCalls, toolCalls...)
	}
}

func (c *streamIterationCollector) content() string {
	return c.visibleContent.String()
}

func (c *streamIterationCollector) outcome(totalTokens int) iterationOutcome {
	return iterationOutcome{
		content:     c.content(),
		totalTokens: totalTokens,
		traceItems:  c.traceItems,
	}
}

func (a *ReActAgent) maxToolIterationsError() error {
	return fmt.Errorf("已达到最大工具迭代次数 (%d)", a.maxToolIterations)
}

func (a *ReActAgent) failStreamRun(callback StreamCallback, iteration int, err error) error {
	if callback != nil {
		callback(StreamChunk{Error: err, Iteration: iteration})
	}
	return err
}

func (a *ReActAgent) runIterationLoop(
	ctx context.Context,
	msg bus.InboundMessage,
	modelName string,
	messages []providers.ChatMessage,
	step func(start *iterationStart, state *llmLoopState) (string, bool, error),
	onMaxIterations func(state *llmLoopState) error,
) (*llmLoopState, string, error) {
	state := &llmLoopState{
		currentMessages: messages,
	}

	for state.iteration < a.maxToolIterations {
		state.iteration++
		start, err := a.beginLLMIteration(ctx, msg, modelName, state.currentMessages)
		if err != nil {
			return state, "", err
		}
		state.currentMessages = start.messages

		content, done, err := step(start, state)
		if err != nil {
			return state, "", err
		}
		if done {
			return state, content, nil
		}
	}

	if onMaxIterations == nil {
		return state, "", a.maxToolIterationsError()
	}

	return state, "", onMaxIterations(state)
}

func (a *ReActAgent) collectSyncIteration(
	ctx context.Context,
	provider providers.Provider,
	request providers.ChatRequest,
	iteration int,
	traceItems []MessageTraceItem,
) (*iterationOutcome, error) {
	resp, err := provider.Chat(ctx, request)
	if err != nil {
		return nil, fmt.Errorf(errSummaryLLMRequestFailed, err)
	}

	contentExtraction := extractThinkBlocks(resp.Content)
	reasoningExtraction := extractThinkBlocks(resp.Reasoning)
	traceItems = a.appendThinkingTrace(traceItems, reasoningExtraction.Reasoning, iteration)
	traceItems = a.appendThinkingTrace(traceItems, contentExtraction.Reasoning, iteration)

	return &iterationOutcome{
		content:     contentExtraction.Visible,
		toolCalls:   resp.ToolCalls,
		totalTokens: resp.Usage.TotalTokens,
		traceItems:  traceItems,
	}, nil
}

func (a *ReActAgent) collectStreamIteration(
	ctx context.Context,
	provider providers.Provider,
	request providers.ChatRequest,
	iteration int,
	callback StreamCallback,
) (*streamIterationCollector, error) {
	collector := &streamIterationCollector{}
	filter := newThinkStreamFilter()

	if err := provider.ChatStream(ctx, request, a.ChunkCallback(filter, iteration, collector, callback)); err != nil {
		if callback != nil {
			callback(StreamChunk{Error: err, Iteration: iteration})
		}
		return nil, fmt.Errorf(errSummaryLLMRequestFailed, err)
	}

	flushed := filter.Flush()
	collector.appendVisible(flushed.Visible)
	flushedReasoning := joinReasoningParts(flushed.Reasoning)
	collector.appendReasoning(a, flushedReasoning, iteration)
	if callback != nil && (flushed.Visible != "" || flushedReasoning != "") {
		if err := callback(StreamChunk{
			Iteration: iteration,
			Content:   flushed.Visible,
			Reasoning: flushedReasoning,
		}); err != nil {
			return nil, err
		}
	}

	return collector, nil
}

func (a *ReActAgent) resolveStreamCollectorOutcome(
	collector *streamIterationCollector,
	totalTokens int,
) iterationOutcome {
	if collector == nil {
		return iterationOutcome{totalTokens: totalTokens}
	}

	outcome := collector.outcome(totalTokens)
	if len(collector.toolCalls) == 0 {
		return outcome
	}

	mergedToolCalls := a.mergeToolCalls(collector.toolCalls)
	outcome.toolCalls = a.validateToolCalls(mergedToolCalls)
	return outcome
}

func (a *ReActAgent) applyStreamIterationResolution(
	ctx context.Context,
	msg bus.InboundMessage,
	sessionKey string,
	state *llmLoopState,
	outcome iterationOutcome,
	callback StreamCallback,
) (*iterationResolution, error) {
	resolution, assistantMessageID, err := a.resolveStreamIteration(
		ctx,
		msg,
		state.currentMessages,
		sessionKey,
		outcome,
		state.iteration,
		callback,
	)
	if err != nil {
		return nil, err
	}

	state.currentMessages = resolution.messages
	state.traceItems = resolution.traceItems
	state.assistantMessageID = assistantMessageID

	return resolution, nil
}
