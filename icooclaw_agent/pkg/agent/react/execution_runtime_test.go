package react

import (
	"context"
	"errors"
	"strings"
	"testing"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/tools"
)

func TestRunIterationLoop_CompletesBeforeMaxIterations(t *testing.T) {
	hooks := &countingHooks{}
	agent := &ReActAgent{
		tools:             tools.NewRegistry(),
		logger:            newTestAgentWithLogger().logger,
		hooks:             hooks,
		maxToolIterations: 3,
	}

	var seenIterations []int
	state, content, err := agent.runIterationLoop(context.Background(), bus.InboundMessage{
		Channel:   consts.WEBSOCKET,
		SessionID: "loop-complete",
		Text:      "hello",
	}, "test-model", []providers.ChatMessage{
		{Role: consts.RoleUser.ToString(), Content: "hello"},
	}, func(start *iterationStart, state *llmLoopState) (string, bool, error) {
		seenIterations = append(seenIterations, state.iteration)
		if len(start.request.Messages) == 0 {
			t.Fatal("expected request messages in loop step")
		}
		if state.iteration == 2 {
			return "done", true, nil
		}
		return "", false, nil
	}, nil)
	if err != nil {
		t.Fatalf("runIterationLoop() error = %v", err)
	}

	if content != "done" {
		t.Fatalf("content = %q, want %q", content, "done")
	}
	if state.iteration != 2 {
		t.Fatalf("iteration = %d, want %d", state.iteration, 2)
	}
	if len(seenIterations) != 2 {
		t.Fatalf("seen iteration count = %d, want %d", len(seenIterations), 2)
	}
	if hooks.beforeCount != 2 {
		t.Fatalf("beforeCount = %d, want %d", hooks.beforeCount, 2)
	}
}

func TestRunIterationLoop_CallsOnMaxIterations(t *testing.T) {
	agent := &ReActAgent{
		tools:             tools.NewRegistry(),
		logger:            newTestAgentWithLogger().logger,
		maxToolIterations: 2,
	}

	var onMaxCalled bool
	maxErr := errors.New("max reached")
	state, content, err := agent.runIterationLoop(context.Background(), bus.InboundMessage{
		Channel:   consts.WEBSOCKET,
		SessionID: "loop-max",
		Text:      "hello",
	}, "test-model", []providers.ChatMessage{
		{Role: consts.RoleUser.ToString(), Content: "hello"},
	}, func(start *iterationStart, state *llmLoopState) (string, bool, error) {
		return "", false, nil
	}, func(state *llmLoopState) error {
		onMaxCalled = true
		return maxErr
	})
	if !errors.Is(err, maxErr) {
		t.Fatalf("error = %v, want %v", err, maxErr)
	}
	if !onMaxCalled {
		t.Fatal("expected onMaxIterations callback to be called")
	}
	if content != "" {
		t.Fatalf("content = %q, want empty", content)
	}
	if state.iteration != 2 {
		t.Fatalf("iteration = %d, want %d", state.iteration, 2)
	}
}

func TestCollectStreamIteration_CollectsVisibleReasoningAndToolCalls(t *testing.T) {
	provider := &mockProvider{
		name:         "mock",
		defaultModel: "test-model",
		streamRuns: [][]mockStreamEvent{
			{
				{content: "<think>内部"},
				{content: "</think>外"},
				{content: "部"},
				{reasoning: "结构化"},
				{
					toolCalls: []providers.ToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							}{
								Name:      "lookup",
								Arguments: `{"q":"天气"}`,
							},
						},
					},
				},
			},
		},
	}
	agent := newTestAgentWithLogger()

	var chunks []StreamChunk
	collector, err := agent.collectStreamIteration(context.Background(), provider, providers.ChatRequest{
		Model: "test-model",
		Messages: []providers.ChatMessage{
			{Role: consts.RoleUser.ToString(), Content: "hello"},
		},
	}, 1, func(chunk StreamChunk) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("collectStreamIteration() error = %v", err)
	}

	if collector.content() != "外部" {
		t.Fatalf("collector content = %q, want %q", collector.content(), "外部")
	}
	if len(collector.toolCalls) != 1 {
		t.Fatalf("tool call count = %d, want %d", len(collector.toolCalls), 1)
	}
	if len(collector.traceItems) != 1 {
		t.Fatalf("trace item count = %d, want %d", len(collector.traceItems), 1)
	}
	if !strings.Contains(collector.traceItems[0].Content, "内部") || !strings.Contains(collector.traceItems[0].Content, "结构化") {
		t.Fatalf("trace reasoning = %q, want to contain internal and structured reasoning", collector.traceItems[0].Content)
	}
	if len(chunks) < 2 {
		t.Fatalf("chunk count = %d, want at least %d", len(chunks), 2)
	}
	if chunks[len(chunks)-1].Content != "外部" {
		t.Fatalf("final chunk content = %q, want %q", chunks[len(chunks)-1].Content, "外部")
	}
}

func TestResolveIterationOutcome_WithToolCallsContinuesLoop(t *testing.T) {
	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(mockTool{
		name:   "lookup",
		result: "工具结果",
	})

	agent := &ReActAgent{
		tools:  toolRegistry,
		logger: newTestAgentWithLogger().logger,
	}

	resolution, err := agent.resolveIterationOutcome(context.Background(), bus.InboundMessage{
		Channel:   consts.WEBSOCKET,
		SessionID: "resolve-tools",
	}, []providers.ChatMessage{
		{Role: consts.RoleUser.ToString(), Content: "hello"},
	}, iterationOutcome{
		content: "",
		toolCalls: []providers.ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      "lookup",
					Arguments: `{"q":"天气"}`,
				},
			},
		},
		totalTokens: 11,
	}, 1, nil)
	if err != nil {
		t.Fatalf("resolveIterationOutcome() error = %v", err)
	}

	if !resolution.continueLoop {
		t.Fatal("expected continueLoop to be true")
	}
	if len(resolution.messages) != 3 {
		t.Fatalf("message count = %d, want %d", len(resolution.messages), 3)
	}
	if resolution.messages[1].Role != consts.RoleAssistant.ToString() {
		t.Fatalf("assistant role = %q, want %q", resolution.messages[1].Role, consts.RoleAssistant.ToString())
	}
	if len(resolution.messages[1].ToolCalls) != 1 {
		t.Fatalf("assistant tool call count = %d, want %d", len(resolution.messages[1].ToolCalls), 1)
	}
	if resolution.messages[2].Role != consts.RoleTool.ToString() {
		t.Fatalf("tool role = %q, want %q", resolution.messages[2].Role, consts.RoleTool.ToString())
	}
	if resolution.messages[2].Content != "工具结果" {
		t.Fatalf("tool content = %q, want %q", resolution.messages[2].Content, "工具结果")
	}
	if resolution.messages[2].TotalTokens != 11 {
		t.Fatalf("tool total_tokens = %d, want %d", resolution.messages[2].TotalTokens, 11)
	}
	if len(resolution.traceItems) != 1 {
		t.Fatalf("trace item count = %d, want %d", len(resolution.traceItems), 1)
	}
	if resolution.traceItems[0].ToolResult != "工具结果" {
		t.Fatalf("trace tool result = %q, want %q", resolution.traceItems[0].ToolResult, "工具结果")
	}
}

func TestResolveIterationOutcome_WithoutToolCallsSendsDone(t *testing.T) {
	agent := newTestAgentWithLogger()

	var chunks []StreamChunk
	resolution, err := agent.resolveIterationOutcome(context.Background(), bus.InboundMessage{
		Channel:   consts.WEBSOCKET,
		SessionID: "resolve-final",
	}, []providers.ChatMessage{
		{Role: consts.RoleUser.ToString(), Content: "hello"},
	}, iterationOutcome{
		content:     "最终答案",
		totalTokens: 22,
		traceItems: []MessageTraceItem{
			{Type: "thinking", Content: "分析"},
		},
	}, 1, func(chunk StreamChunk) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("resolveIterationOutcome() error = %v", err)
	}

	if resolution.continueLoop {
		t.Fatal("expected continueLoop to be false")
	}
	if len(resolution.messages) != 2 {
		t.Fatalf("message count = %d, want %d", len(resolution.messages), 2)
	}
	if resolution.messages[1].Content != "最终答案" {
		t.Fatalf("assistant content = %q, want %q", resolution.messages[1].Content, "最终答案")
	}
	if len(chunks) != 1 || !chunks[0].Done {
		t.Fatalf("chunks = %+v, want one done chunk", chunks)
	}
	if chunks[0].TotalTokens != 22 {
		t.Fatalf("done total_tokens = %d, want %d", chunks[0].TotalTokens, 22)
	}
}
