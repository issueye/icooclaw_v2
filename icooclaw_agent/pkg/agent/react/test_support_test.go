package react

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
)

// Mock provider for testing
type mockProvider struct {
	name           string
	defaultModel   string
	chatResponses  []*providers.ChatResponse
	chatErr        error
	chatIndex      int
	chatRequests   []providers.ChatRequest
	streamRuns     [][]mockStreamEvent
	streamErr      error
	streamIndex    int
	streamRequests []providers.ChatRequest
}

type mockStreamEvent struct {
	content   string
	reasoning string
	toolCalls []providers.ToolCall
	done      bool
}

func (m *mockProvider) GetName() string {
	return m.name
}

func (m *mockProvider) GetModel() string {
	return m.defaultModel
}

func (m *mockProvider) SetModel(model string) {
	m.defaultModel = model
}

func (m *mockProvider) Chat(ctx context.Context, req providers.ChatRequest) (*providers.ChatResponse, error) {
	m.chatRequests = append(m.chatRequests, req)
	if m.chatErr != nil {
		return nil, m.chatErr
	}
	if m.chatIndex < len(m.chatResponses) {
		resp := m.chatResponses[m.chatIndex]
		m.chatIndex++
		return resp, nil
	}
	return &providers.ChatResponse{
		Content: "mock response",
	}, nil
}

func (m *mockProvider) ChatStream(ctx context.Context, req providers.ChatRequest, callback providers.StreamCallback) error {
	m.streamRequests = append(m.streamRequests, req)
	if m.streamErr != nil {
		return m.streamErr
	}
	if m.streamIndex >= len(m.streamRuns) {
		return nil
	}
	run := m.streamRuns[m.streamIndex]
	m.streamIndex++
	for _, event := range run {
		if err := callback(event.content, event.reasoning, event.toolCalls, event.done); err != nil {
			return err
		}
	}
	return nil
}

type mockTool struct {
	name   string
	result string
}

type countingHooks struct {
	mu               sync.Mutex
	beforeCount      int
	afterCount       int
	agentStart       int
	agentEnd         int
	beforeAppendText string
}

func (h *countingHooks) OnGetProvider(ctx context.Context, defaultModel string, storage *storage.ProviderStorage) error {
	return nil
}

func (h *countingHooks) OnCreateAgent(ctx context.Context, a *ReActAgent) (*ReActAgent, error) {
	return a, nil
}

func (h *countingHooks) OnBuildMessagesBefore(ctx context.Context, sessionKey string, msg bus.InboundMessage, history []providers.ChatMessage) ([]providers.ChatMessage, error) {
	return history, nil
}

func (h *countingHooks) OnBuildMessagesAfter(ctx context.Context, sessionKey string, msg bus.InboundMessage, history []providers.ChatMessage) ([]providers.ChatMessage, error) {
	return history, nil
}

func (h *countingHooks) OnRunLLMBefore(ctx context.Context, msg bus.InboundMessage, history []providers.ChatMessage) ([]providers.ChatMessage, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.beforeCount++
	if h.beforeAppendText != "" {
		history = append(history, providers.ChatMessage{
			Role:    consts.RoleSystem.ToString(),
			Content: h.beforeAppendText,
		})
	}
	return history, nil
}

func (h *countingHooks) OnRunLLMAfter(ctx context.Context, msg bus.InboundMessage, history []providers.ChatMessage) ([]providers.ChatMessage, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.afterCount++
	return history, nil
}

func (h *countingHooks) OnToolCallBefore(ctx context.Context, toolName string, tc providers.ToolCall, msg bus.InboundMessage) (providers.ToolCall, error) {
	return tc, nil
}

func (h *countingHooks) OnToolCallAfter(ctx context.Context, toolName string, msg bus.InboundMessage, result *tools.Result) error {
	return nil
}

func (h *countingHooks) OnToolParseArguments(ctx context.Context, toolName string, tc providers.ToolCall, msg bus.InboundMessage) (map[string]any, error) {
	var args map[string]any
	if tc.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, err
		}
	}
	return args, nil
}

func (h *countingHooks) OnAgentStart(ctx context.Context, msg bus.InboundMessage) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.agentStart++
	return nil
}

func (h *countingHooks) OnAgentEnd(ctx context.Context, msg bus.InboundMessage, response string, iteration int, messageID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.agentEnd++
	return nil
}

func (m mockTool) Name() string {
	return m.name
}

func (m mockTool) Description() string {
	return "mock tool"
}

func (m mockTool) Parameters() map[string]any {
	return map[string]any{
		"q": map[string]any{"type": "string"},
	}
}

func (m mockTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	return tools.SuccessResult(m.result)
}

func newTestStorageForReact(t *testing.T) *storage.Storage {
	t.Helper()

	workspaceDir := t.TempDir()
	for _, name := range []string{"SOUL.md", "USER.md"} {
		if err := os.WriteFile(filepath.Join(workspaceDir, name), []byte("# test\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
	}
	dbPath := filepath.Join(t.TempDir(), "react.db")
	store, err := storage.New(workspaceDir, "", dbPath)
	if err != nil {
		t.Fatalf("storage.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}

func newTestAgentWithLogger() *ReActAgent {
	return &ReActAgent{
		tools:  tools.NewRegistry(),
		logger: slog.Default(),
	}
}
