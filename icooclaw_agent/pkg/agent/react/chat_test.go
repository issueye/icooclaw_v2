package react

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"unicode/utf8"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/memory"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/skill"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
)

func TestStreamChunk_Types(t *testing.T) {
	chunks := []StreamChunk{
		{Content: "test content"},
		{Reasoning: "thinking..."},
		{ToolName: "search"},
		{ToolResult: "result"},
		{Iteration: 1},
		{Done: true},
		{Error: errors.New("test error")},
	}

	for i, chunk := range chunks {
		if chunk.Content == "" && chunk.Reasoning == "" && chunk.ToolName == "" && chunk.ToolResult == "" && !chunk.Done && chunk.Error == nil && chunk.Iteration == 0 {
			t.Errorf("chunk %d should have some data", i)
		}
	}
}

func TestStreamCallback(t *testing.T) {
	var receivedChunks []StreamChunk
	callback := func(chunk StreamChunk) error {
		receivedChunks = append(receivedChunks, chunk)
		return nil
	}

	// Test callback invocation
	_ = callback(StreamChunk{Content: "hello"})
	_ = callback(StreamChunk{Done: true})

	if len(receivedChunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(receivedChunks))
	}
	if receivedChunks[0].Content != "hello" {
		t.Errorf("expected 'hello', got '%s'", receivedChunks[0].Content)
	}
	if !receivedChunks[1].Done {
		t.Error("second chunk should be done")
	}
}

func TestReActAgent_ChatStream_NoProvider(t *testing.T) {
	agent, err := NewReActAgent(context.Background(), nil, Dependencies{})
	if err != nil {
		t.Error("failed to create agent")
	}

	var receivedChunks []StreamChunk
	callback := func(chunk StreamChunk) error {
		receivedChunks = append(receivedChunks, chunk)
		return nil
	}

	msg := bus.InboundMessage{
		Channel:   "test",
		SessionID: "session-1",
		Text:      "hello",
	}

	_, _, err = agent.ChatStream(context.Background(), msg, callback)
	if err == nil {
		t.Error("expected error without provider")
	}

	// Should receive error chunk
	if len(receivedChunks) == 0 {
		t.Error("expected error chunk")
	}
	if receivedChunks[0].Error == nil {
		t.Error("chunk should have error")
	}
}

func TestStripThinkBlocks(t *testing.T) {
	input := "<think>\ninternal\n</think>\n\n最终答案"
	if got := stripThinkBlocks(input); got != "最终答案" {
		t.Fatalf("stripThinkBlocks() = %q, want %q", got, "最终答案")
	}
}

func TestExtractThinkBlocks(t *testing.T) {
	input := "<think>分析一</think>\n最终答案\n<think>分析二</think>"
	got := extractThinkBlocks(input)
	if got.Visible != "最终答案" {
		t.Fatalf("visible = %q, want %q", got.Visible, "最终答案")
	}
	if got.Reasoning != "分析一\n\n分析二" {
		t.Fatalf("reasoning = %q, want %q", got.Reasoning, "分析一\n\n分析二")
	}
}

func TestThinkStreamFilter(t *testing.T) {
	filter := newThinkStreamFilter()
	var visible string
	var reasoning string
	part := filter.Push("<thi")
	visible += part.Visible
	reasoning += part.Reasoning
	part = filter.Push("nk>internal")
	visible += part.Visible
	reasoning += part.Reasoning
	part = filter.Push("</thi")
	visible += part.Visible
	reasoning += part.Reasoning
	part = filter.Push("nk>外部")
	visible += part.Visible
	reasoning += part.Reasoning
	part = filter.Flush()
	visible += part.Visible
	reasoning += part.Reasoning

	if visible != "外部" {
		t.Fatalf("stream visible = %q, want %q", visible, "外部")
	}
	if reasoning != "internal" {
		t.Fatalf("stream reasoning = %q, want %q", reasoning, "internal")
	}
}

func TestThinkStreamFilter_PreservesUTF8Boundaries(t *testing.T) {
	filter := newThinkStreamFilter()
	var visible string

	part := filter.Push("我是**子任务")
	visible += part.Visible
	part = filter.Flush()
	visible += part.Visible

	if visible != "我是**子任务" {
		t.Fatalf("stream visible = %q, want %q", visible, "我是**子任务")
	}
	if !utf8.ValidString(visible) {
		t.Fatalf("stream visible should be valid UTF-8, got %q", visible)
	}
}

func TestSaveAssistantMessage_StripsThinkContentAndTraceResults(t *testing.T) {
	store := newTestStorageForReact(t)
	agent := &ReActAgent{storage: store, logger: slog.Default()}
	traceItems := []MessageTraceItem{
		{Type: "thinking", Content: "<think>analysis</think>"},
		{Type: "tool_call", ToolCallID: "call_1", ToolResult: "<think>debug</think>\n\n可见结果"},
	}

	msgID, err := agent.saveAssistantMessage(context.Background(), "s1", "<think>internal</think>\n\n用户可见", traceItems, 2, 128)
	if err != nil {
		t.Fatalf("saveAssistantMessage() error = %v", err)
	}

	saved, err := store.Message().GetByID(msgID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if saved.Content != "用户可见" {
		t.Fatalf("saved content = %q, want %q", saved.Content, "用户可见")
	}
	if saved.Thinking != "analysis" {
		t.Fatalf("saved thinking = %q, want %q", saved.Thinking, "analysis")
	}
	if saved.TotalTokens != 128 {
		t.Fatalf("saved total_tokens = %d, want %d", saved.TotalTokens, 128)
	}

	var meta AssistantMessageMetadata
	if err := json.Unmarshal([]byte(saved.Metadata), &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if len(meta.TraceItems) != 2 {
		t.Fatalf("trace item count = %d, want 2", len(meta.TraceItems))
	}
	if meta.ReasoningContent != "analysis" {
		t.Fatalf("reasoning_content = %q, want %q", meta.ReasoningContent, "analysis")
	}
	if meta.Iteration != 2 {
		t.Fatalf("iteration = %d, want %d", meta.Iteration, 2)
	}
	if meta.TraceItems[1].ToolResult != "可见结果" {
		t.Fatalf("tool_result = %q, want %q", meta.TraceItems[1].ToolResult, "可见结果")
	}
}

func TestReActAgent_ChatStream_PersistsAssistantMessagePerIteration(t *testing.T) {
	store := newTestStorageForReact(t)
	if err := store.Param().SaveOrUpdateByKey(&storage.ParamConfig{
		Key:     consts.DEFAULT_MODEL_KEY,
		Value:   "mock/test-model",
		Enabled: true,
	}); err != nil {
		t.Fatalf("save default model: %v", err)
	}

	provider := &mockProvider{
		name:         "mock",
		defaultModel: "test-model",
		streamRuns: [][]mockStreamEvent{
			{
				{
					reasoning: "先分析问题",
				},
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
			{
				{
					reasoning: "整理工具结果",
				},
				{
					content: "最终",
				},
				{
					content: "答案",
				},
			},
		},
	}

	providerManager := providers.NewManager(store, nil)
	providerManager.Register("mock", provider)

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(mockTool{
		name:   "lookup",
		result: "<think>工具内部分析</think>\n\n工具可见结果",
	})

	agent, err := NewReActAgent(context.Background(), nil, Dependencies{
		Tools:             toolRegistry,
		Memory:            memory.NewLoader(store, 100, slog.Default()),
		Skills:            skill.NewLoader(t.TempDir(), store, slog.Default()),
		Storage:           store,
		ProviderManager:   providerManager,
		Logger:            slog.Default(),
		MaxToolIterations: 4,
	})
	if err != nil {
		t.Fatalf("NewReActAgent() error = %v", err)
	}

	var chunks []StreamChunk
	content, iteration, err := agent.ChatStream(context.Background(), bus.InboundMessage{
		Channel:   consts.WEBSOCKET,
		SessionID: "session-stream-iterations",
		Text:      "帮我查一下天气并总结",
		Sender: bus.SenderInfo{
			ID: "user-1",
		},
	}, func(chunk StreamChunk) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}

	if content != "最终答案" {
		t.Fatalf("content = %q, want %q", content, "最终答案")
	}
	if iteration != 2 {
		t.Fatalf("iteration = %d, want %d", iteration, 2)
	}

	sessionKey := consts.GetSessionKey(consts.WEBSOCKET, "session-stream-iterations")
	page, err := store.Message().Page(&storage.QueryMessage{SessionID: sessionKey})
	if err != nil {
		t.Fatalf("Page() error = %v", err)
	}
	if len(page.Records) != 3 {
		t.Fatalf("message count = %d, want %d", len(page.Records), 3)
	}

	var assistantMessages []storage.Message
	for _, record := range page.Records {
		if record.Role == consts.RoleAssistant {
			assistantMessages = append(assistantMessages, record)
		}
	}
	if len(assistantMessages) != 2 {
		t.Fatalf("assistant message count = %d, want %d", len(assistantMessages), 2)
	}

	assistantByIteration := make(map[int]storage.Message, len(assistantMessages))
	for _, record := range assistantMessages {
		var meta AssistantMessageMetadata
		if err := json.Unmarshal([]byte(record.Metadata), &meta); err != nil {
			t.Fatalf("unmarshal assistant metadata: %v", err)
		}
		assistantByIteration[meta.Iteration] = record
	}

	firstRecord, ok := assistantByIteration[1]
	if !ok {
		t.Fatal("missing iteration 1 assistant message")
	}
	var firstMeta AssistantMessageMetadata
	if err := json.Unmarshal([]byte(firstRecord.Metadata), &firstMeta); err != nil {
		t.Fatalf("unmarshal first metadata: %v", err)
	}
	if firstRecord.Content != "" {
		t.Fatalf("first iteration content = %q, want empty", firstRecord.Content)
	}
	if firstMeta.Iteration != 1 {
		t.Fatalf("first iteration meta = %d, want %d", firstMeta.Iteration, 1)
	}
	if len(firstMeta.TraceItems) != 3 {
		t.Fatalf("first trace count = %d, want %d", len(firstMeta.TraceItems), 3)
	}
	if firstMeta.TraceItems[1].ToolCallID != "call_1" {
		t.Fatalf("first tool_call_id = %q, want %q", firstMeta.TraceItems[1].ToolCallID, "call_1")
	}
	if firstMeta.TraceItems[1].ToolResult != "工具可见结果" {
		t.Fatalf("first tool_result = %q, want %q", firstMeta.TraceItems[1].ToolResult, "工具可见结果")
	}

	secondRecord, ok := assistantByIteration[2]
	if !ok {
		t.Fatal("missing iteration 2 assistant message")
	}
	var secondMeta AssistantMessageMetadata
	if err := json.Unmarshal([]byte(secondRecord.Metadata), &secondMeta); err != nil {
		t.Fatalf("unmarshal second metadata: %v", err)
	}
	if secondRecord.Content != "最终答案" {
		t.Fatalf("second iteration content = %q, want %q", secondRecord.Content, "最终答案")
	}
	if secondMeta.Iteration != 2 {
		t.Fatalf("second iteration meta = %d, want %d", secondMeta.Iteration, 2)
	}
	if secondMeta.ReasoningContent != "整理工具结果" {
		t.Fatalf("second reasoning_content = %q, want %q", secondMeta.ReasoningContent, "整理工具结果")
	}

	if len(chunks) == 0 || !chunks[len(chunks)-1].Done {
		t.Fatal("expected final done chunk")
	}
}

func TestExecuteToolCalls_SharedSemanticsForSyncAndStream(t *testing.T) {
	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(mockTool{
		name:   "lookup",
		result: "<think>工具推理</think>\n\n工具可见结果",
	})

	agent := &ReActAgent{
		tools:  toolRegistry,
		logger: slog.Default(),
	}

	toolCalls := []providers.ToolCall{
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
	}

	syncMsgs, syncTrace, err := agent.executeToolCalls(context.Background(), bus.InboundMessage{
		Channel:   consts.WEBSOCKET,
		SessionID: "sync",
	}, toolCalls, 1, nil)
	if err != nil {
		t.Fatalf("executeToolCalls(sync) error = %v", err)
	}

	var streamChunks []StreamChunk
	streamMsgs, streamTrace, err := agent.executeToolCalls(context.Background(), bus.InboundMessage{
		Channel:   consts.WEBSOCKET,
		SessionID: "stream",
	}, toolCalls, 1, func(chunk StreamChunk) error {
		streamChunks = append(streamChunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("executeToolCalls(stream) error = %v", err)
	}

	if len(syncMsgs) != 1 || len(streamMsgs) != 1 {
		t.Fatalf("message counts = %d/%d, want 1/1", len(syncMsgs), len(streamMsgs))
	}
	if syncMsgs[0].Content != "工具可见结果" || streamMsgs[0].Content != "工具可见结果" {
		t.Fatalf("visible results = %q/%q, want %q", syncMsgs[0].Content, streamMsgs[0].Content, "工具可见结果")
	}
	if len(syncTrace) != 2 || len(streamTrace) != 2 {
		t.Fatalf("trace counts = %d/%d, want 2/2", len(syncTrace), len(streamTrace))
	}
	if syncTrace[0].Type != "tool_call" || streamTrace[0].Type != "tool_call" {
		t.Fatalf("first trace types = %q/%q, want tool_call", syncTrace[0].Type, streamTrace[0].Type)
	}
	if syncTrace[0].ToolResult != "工具可见结果" || streamTrace[0].ToolResult != "工具可见结果" {
		t.Fatalf("tool results in trace = %q/%q, want %q", syncTrace[0].ToolResult, streamTrace[0].ToolResult, "工具可见结果")
	}
	if syncTrace[1].Type != "thinking" || streamTrace[1].Type != "thinking" {
		t.Fatalf("second trace types = %q/%q, want thinking", syncTrace[1].Type, streamTrace[1].Type)
	}
	if len(streamChunks) != 3 {
		t.Fatalf("stream chunk count = %d, want %d", len(streamChunks), 3)
	}
	if streamChunks[0].ToolName != "lookup" {
		t.Fatalf("first chunk tool name = %q, want %q", streamChunks[0].ToolName, "lookup")
	}
	if streamChunks[1].Reasoning != "工具推理" {
		t.Fatalf("second chunk reasoning = %q, want %q", streamChunks[1].Reasoning, "工具推理")
	}
	if streamChunks[2].ToolResult != "工具可见结果" {
		t.Fatalf("third chunk tool result = %q, want %q", streamChunks[2].ToolResult, "工具可见结果")
	}
}

func TestRunLLMHooks_CountsMatchBetweenSyncAndStream(t *testing.T) {
	msg := bus.InboundMessage{
		Channel:   consts.WEBSOCKET,
		SessionID: "hooks-session",
		Text:      "帮我查天气",
	}
	toolCall := providers.ToolCall{
		ID:   "call_1",
		Type: "function",
		Function: struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		}{
			Name:      "lookup",
			Arguments: `{"q":"天气"}`,
		},
	}

	syncHooks := &countingHooks{}
	streamHooks := &countingHooks{}

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(mockTool{
		name:   "lookup",
		result: "工具结果",
	})

	syncAgent := &ReActAgent{
		tools:             toolRegistry,
		logger:            slog.Default(),
		hooks:             syncHooks,
		maxToolIterations: 4,
	}
	streamAgent := &ReActAgent{
		tools:             toolRegistry,
		logger:            slog.Default(),
		hooks:             streamHooks,
		maxToolIterations: 4,
	}

	syncProvider := &mockProvider{
		name:         "mock",
		defaultModel: "test-model",
		chatResponses: []*providers.ChatResponse{
			{ToolCalls: []providers.ToolCall{toolCall}},
			{Content: "最终答案"},
		},
	}
	streamProvider := &mockProvider{
		name:         "mock",
		defaultModel: "test-model",
		streamRuns: [][]mockStreamEvent{
			{{toolCalls: []providers.ToolCall{toolCall}}},
			{{content: "最终答案"}},
		},
	}

	if _, _, _, _, err := syncAgent.RunLLM(context.Background(), "test-model", syncProvider, []providers.ChatMessage{{Role: "user", Content: "hi"}}, msg); err != nil {
		t.Fatalf("RunLLM() error = %v", err)
	}
	if _, _, _, _, err := streamAgent.RunLLMStream(context.Background(), "web:hooks-session", "test-model", streamProvider, []providers.ChatMessage{{Role: "user", Content: "hi"}}, msg, nil); err != nil {
		t.Fatalf("RunLLMStream() error = %v", err)
	}

	if syncHooks.beforeCount != 2 || streamHooks.beforeCount != 2 {
		t.Fatalf("before counts = %d/%d, want 2/2", syncHooks.beforeCount, streamHooks.beforeCount)
	}
	if syncHooks.afterCount != 2 || streamHooks.afterCount != 2 {
		t.Fatalf("after counts = %d/%d, want 2/2", syncHooks.afterCount, streamHooks.afterCount)
	}
}

func TestChat_SanitizesDirtyHistoryBeforeProviderRequest(t *testing.T) {
	store := newTestStorageForReact(t)
	if err := store.Param().SaveOrUpdateByKey(&storage.ParamConfig{
		Key:     consts.DEFAULT_MODEL_KEY,
		Value:   "mock/test-model",
		Enabled: true,
	}); err != nil {
		t.Fatalf("save default model: %v", err)
	}

	provider := &mockProvider{
		name:         "mock",
		defaultModel: "test-model",
		chatResponses: []*providers.ChatResponse{
			{Content: "正常回复"},
		},
	}
	providerManager := providers.NewManager(store, nil)
	providerManager.Register("mock", provider)

	sessionKey := consts.GetSessionKey(consts.WEBSOCKET, "sanitize-history")
	for _, message := range []*storage.Message{
		{
			SessionID: sessionKey,
			Role:      consts.RoleUser,
			Content:   "旧问题",
		},
		{
			SessionID: sessionKey,
			Role:      consts.RoleAssistant,
			Content:   "",
			Thinking:  "内部思考",
			Metadata:  `{"type":"assistant_trace","reasoning_content":"内部思考"}`,
		},
		{
			SessionID: sessionKey,
			Role:      consts.RoleAssistant,
			Content:   "历史答案",
		},
	} {
		if err := store.Message().Save(message); err != nil {
			t.Fatalf("save message: %v", err)
		}
	}

	agent, err := NewReActAgent(context.Background(), nil, Dependencies{
		Tools:             tools.NewRegistry(),
		Memory:            memory.NewLoader(store, 100, slog.Default()),
		Skills:            skill.NewLoader(t.TempDir(), store, slog.Default()),
		Storage:           store,
		ProviderManager:   providerManager,
		Logger:            slog.Default(),
		MaxToolIterations: 2,
	})
	if err != nil {
		t.Fatalf("NewReActAgent() error = %v", err)
	}

	if _, _, err := agent.Chat(context.Background(), bus.InboundMessage{
		Channel:   consts.WEBSOCKET,
		SessionID: "sanitize-history",
		Text:      "新问题",
	}); err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if len(provider.chatRequests) != 1 {
		t.Fatalf("provider chat request count = %d, want %d", len(provider.chatRequests), 1)
	}

	for _, message := range provider.chatRequests[0].Messages {
		if strings.TrimSpace(message.Content) == "" && len(message.ToolCalls) == 0 {
			t.Fatalf("dirty message should not be sent to provider: %+v", message)
		}
	}
}

func TestRunLLMStream_RequestIncludesBeforeHookMessages(t *testing.T) {
	msg := bus.InboundMessage{
		Channel:   consts.WEBSOCKET,
		SessionID: "hooks-request-session",
		Text:      "帮我查天气",
	}
	hooks := &countingHooks{
		beforeAppendText: "hook-added-system-message",
	}
	provider := &mockProvider{
		name:         "mock",
		defaultModel: "test-model",
		streamRuns: [][]mockStreamEvent{
			{{content: "最终答案"}},
		},
	}
	agent := &ReActAgent{
		tools:             tools.NewRegistry(),
		logger:            slog.Default(),
		hooks:             hooks,
		maxToolIterations: 2,
	}

	_, _, _, _, err := agent.RunLLMStream(
		context.Background(),
		"web:hooks-request-session",
		"test-model",
		provider,
		[]providers.ChatMessage{{Role: consts.RoleUser.ToString(), Content: "原始消息"}},
		msg,
		nil,
	)
	if err != nil {
		t.Fatalf("RunLLMStream() error = %v", err)
	}

	if len(provider.streamRequests) != 1 {
		t.Fatalf("stream request count = %d, want %d", len(provider.streamRequests), 1)
	}
	req := provider.streamRequests[0]
	if len(req.Messages) != 2 {
		t.Fatalf("request message count = %d, want %d", len(req.Messages), 2)
	}
	if req.Messages[1].Content != "hook-added-system-message" {
		t.Fatalf("second request message = %q, want %q", req.Messages[1].Content, "hook-added-system-message")
	}
}
