package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/scheduler"
	"icooclaw/pkg/skill"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
)

func TestSubAgentToolExecute(t *testing.T) {
	workspace := t.TempDir()
	writeWorkspaceFile(t, workspace, "SOUL.md", "# Soul\nStay concise.")
	writeWorkspaceFile(t, workspace, "USER.md", "# User\nFollow instructions.")

	store, err := storage.New(workspace, "", filepath.Join(workspace, "test.db"))
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	defer store.Close()

	if err := store.Param().Save(&storage.ParamConfig{
		Key:   consts.DEFAULT_MODEL_KEY,
		Value: "mock/mock-model",
	}); err != nil {
		t.Fatalf("save default model: %v", err)
	}

	manager := providers.NewManager(store, nil)
	manager.Register("mock", &mockSubAgentProvider{
		response: "subagent final response",
	})

	registry := tools.NewRegistry()
	taskScheduler := scheduler.NewScheduler(store.Task(), nil, nil)
	taskScheduler.RegisterExecutor(scheduler.TaskExecutorSubAgent, NewSubAgentTaskExecutor(
		store,
		manager,
		skill.NewLoader(workspace, store, nil),
		registry,
		nil,
		nil,
	))
	subagentTool := NewSubAgentTool(store, taskScheduler, manager, skill.NewLoader(workspace, store, nil), registry, nil)
	registry.Register(subagentTool)

	result := subagentTool.Execute(
		tools.WithToolContext(context.Background(), "websocket", "session-1"),
		map[string]any{
			"task":    "Summarize the given input",
			"context": "Input: hello world",
		},
	)
	if result.Error != nil {
		t.Fatalf("execute subagent: %v", result.Error)
	}
	if !result.Success {
		t.Fatalf("expected success result")
	}
	if !strings.Contains(result.Content, "Subagent task queued.") {
		t.Fatalf("unexpected result content: %s", result.Content)
	}
	taskID := extractTaskID(result.Content)
	if taskID == "" {
		t.Fatalf("expected task id in result: %s", result.Content)
	}

	eventually(t, 2*time.Second, func() bool {
		task, err := store.Task().GetByID(taskID)
		if err != nil {
			return false
		}
		return task.LastStatus == scheduler.TaskStatusSuccess && strings.Contains(task.Result, "subagent final response")
	})

	task, err := store.Task().GetByID(taskID)
	if err != nil {
		t.Fatalf("load task: %v", err)
	}
	if !strings.Contains(task.Result, "session-1/subagent/") {
		t.Fatalf("expected child session id in stored result: %s", task.Result)
	}
}

func TestSubAgentToolExecuteSyncWaitsForResult(t *testing.T) {
	workspace := t.TempDir()
	writeWorkspaceFile(t, workspace, "SOUL.md", "# Soul\nStay concise.")
	writeWorkspaceFile(t, workspace, "USER.md", "# User\nFollow instructions.")

	store, err := storage.New(workspace, "", filepath.Join(workspace, "test.db"))
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	defer store.Close()

	if err := store.Param().Save(&storage.ParamConfig{
		Key:   consts.DEFAULT_MODEL_KEY,
		Value: "mock/mock-model",
	}); err != nil {
		t.Fatalf("save default model: %v", err)
	}

	manager := providers.NewManager(store, nil)
	manager.Register("mock", &mockSubAgentProvider{
		response: "subagent sync response",
	})

	registry := tools.NewRegistry()
	messageBus := bus.NewMessageBus(bus.DefaultConfig())
	taskScheduler := scheduler.NewScheduler(store.Task(), messageBus, nil)
	taskScheduler.RegisterExecutor(scheduler.TaskExecutorSubAgent, NewSubAgentTaskExecutor(
		store,
		manager,
		skill.NewLoader(workspace, store, nil),
		registry,
		messageBus,
		nil,
	))
	subagentTool := NewSubAgentTool(store, taskScheduler, manager, skill.NewLoader(workspace, store, nil), registry, nil)
	registry.Register(subagentTool)

	result := subagentTool.Execute(
		tools.WithToolContext(context.Background(), "websocket", "session-sync"),
		map[string]any{
			"task":            "Analyze this",
			"context":         "Input: hello sync",
			"mode":            "sync",
			"timeout_seconds": 2,
		},
	)
	if result.Error != nil {
		t.Fatalf("execute sync subagent: %v", result.Error)
	}
	if !result.Success {
		t.Fatalf("expected sync success result")
	}
	if !strings.Contains(result.Content, "subagent sync response") {
		t.Fatalf("unexpected sync result content: %s", result.Content)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if outbound, ok := messageBus.ConsumeOutbound(ctx); ok {
		t.Fatalf("did not expect outbound message in sync mode: %#v", outbound)
	}

	parentSessionKey := consts.GetSessionKey("websocket", "session-sync")
	messages, err := store.Message().Get(parentSessionKey, 10)
	if err != nil {
		t.Fatalf("get parent messages: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("did not expect parent session message in sync mode")
	}
}

func TestSubAgentToolCloneWithoutSelf(t *testing.T) {
	registry := tools.NewRegistry()
	subagentTool := NewSubAgentTool(nil, nil, nil, nil, registry, nil)
	registry.Register(subagentTool)

	clone := registry.CloneWithout(subagentTool.Name())
	if _, ok := clone.GetOK(subagentTool.Name()); ok {
		t.Fatalf("subagent tool should be excluded from clone")
	}
}

func TestSubAgentTaskExecutorPersistsParentMessageAndOutbound(t *testing.T) {
	workspace := t.TempDir()
	writeWorkspaceFile(t, workspace, "SOUL.md", "# Soul\nStay concise.")
	writeWorkspaceFile(t, workspace, "USER.md", "# User\nFollow instructions.")

	store, err := storage.New(workspace, "", filepath.Join(workspace, "test.db"))
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	defer store.Close()

	if err := store.Param().Save(&storage.ParamConfig{
		Key:   consts.DEFAULT_MODEL_KEY,
		Value: "mock/mock-model",
	}); err != nil {
		t.Fatalf("save default model: %v", err)
	}

	manager := providers.NewManager(store, nil)
	manager.Register("mock", &mockSubAgentProvider{
		response: "subagent final response",
	})

	registry := tools.NewRegistry()
	messageBus := bus.NewMessageBus(bus.DefaultConfig())
	taskScheduler := scheduler.NewScheduler(store.Task(), messageBus, nil)
	taskScheduler.RegisterExecutor(scheduler.TaskExecutorSubAgent, NewSubAgentTaskExecutor(
		store,
		manager,
		skill.NewLoader(workspace, store, nil),
		registry,
		messageBus,
		nil,
	))
	subagentTool := NewSubAgentTool(store, taskScheduler, manager, skill.NewLoader(workspace, store, nil), registry, nil)
	registry.Register(subagentTool)

	result := subagentTool.Execute(
		tools.WithToolContext(context.Background(), "websocket", "session-parent"),
		map[string]any{
			"task":    "Research this",
			"context": "Input: hello world",
		},
	)
	if result.Error != nil {
		t.Fatalf("execute subagent: %v", result.Error)
	}

	taskID := extractTaskID(result.Content)
	if taskID == "" {
		t.Fatalf("expected task id in result: %s", result.Content)
	}

	eventually(t, 2*time.Second, func() bool {
		task, err := store.Task().GetByID(taskID)
		if err != nil {
			return false
		}
		return task.LastStatus == scheduler.TaskStatusSuccess
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	outbound, ok := messageBus.ConsumeOutbound(ctx)
	if !ok {
		t.Fatal("expected outbound subagent result")
	}
	if outbound.Channel != "websocket" || outbound.SessionID != "session-parent" {
		t.Fatalf("unexpected outbound target: %#v", outbound)
	}
	if !strings.Contains(outbound.Text, "subagent final response") {
		t.Fatalf("unexpected outbound text: %s", outbound.Text)
	}

	parentSessionKey := consts.GetSessionKey("websocket", "session-parent")
	messages, err := store.Message().Get(parentSessionKey, 10)
	if err != nil {
		t.Fatalf("get parent messages: %v", err)
	}
	if len(messages) == 0 {
		t.Fatal("expected parent session message")
	}
	if !strings.Contains(messages[0].Content, "subagent final response") {
		t.Fatalf("unexpected parent message content: %s", messages[0].Content)
	}
	if !strings.Contains(messages[0].Metadata, "subagent_result") {
		t.Fatalf("expected subagent result metadata, got: %s", messages[0].Metadata)
	}
}

type mockSubAgentProvider struct {
	response string
	lastReq  providers.ChatRequest
}

func (m *mockSubAgentProvider) Chat(ctx context.Context, req providers.ChatRequest) (*providers.ChatResponse, error) {
	m.lastReq = req
	return &providers.ChatResponse{
		Content: m.response,
	}, nil
}

func (m *mockSubAgentProvider) ChatStream(ctx context.Context, req providers.ChatRequest, callback providers.StreamCallback) error {
	return callback(m.response, "", nil, true)
}

func (m *mockSubAgentProvider) GetName() string {
	return "mock"
}

func (m *mockSubAgentProvider) GetModel() string {
	return "mock-model"
}

func (m *mockSubAgentProvider) SetModel(model string) {}

func TestSubAgentTaskExecutor_UsesSelectedOrDefaultAgent(t *testing.T) {
	workspace := t.TempDir()
	writeWorkspaceFile(t, workspace, "SOUL.md", "# Soul\nStay concise.")
	writeWorkspaceFile(t, workspace, "USER.md", "# User\nFollow instructions.")

	store, err := storage.New(workspace, "", filepath.Join(workspace, "test.db"))
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	defer store.Close()

	if err := store.Param().Save(&storage.ParamConfig{
		Key:   consts.DEFAULT_MODEL_KEY,
		Value: "mock/mock-model",
	}); err != nil {
		t.Fatalf("save default model: %v", err)
	}

	selectedAgent := &storage.Agent{
		Name:         "researcher",
		Type:         storage.AgentTypeSubAgent,
		Description:  "does research",
		SystemPrompt: "use evidence only",
		Enabled:      true,
	}
	if err := store.Agent().Save(selectedAgent); err != nil {
		t.Fatalf("save selected agent: %v", err)
	}
	selectedAgent, err = store.Agent().GetByName("researcher")
	if err != nil {
		t.Fatalf("get selected agent: %v", err)
	}
	defaultAgent, err := store.Agent().GetDefault()
	if err != nil {
		t.Fatalf("get default agent: %v", err)
	}

	mockProvider := &mockSubAgentProvider{response: "subagent final response"}
	manager := providers.NewManager(store, nil)
	manager.Register("mock", mockProvider)

	registry := tools.NewRegistry()
	executor := NewSubAgentTaskExecutor(
		store,
		manager,
		skill.NewLoader(workspace, store, nil),
		registry,
		nil,
		nil,
	)

	runTask := func(agentID string) string {
		payload := subAgentTaskPayload{
			Prompt:          "Research this",
			ParentChannel:   "websocket",
			ParentSessionID: "session-parent",
			ChildSessionID:  "session-parent/subagent/test",
			AgentID:         agentID,
		}
		data, _ := json.Marshal(payload)
		_, err := executor(context.Background(), &scheduler.Task{
			ID:     "task-test",
			Params: string(data),
		})
		if err != nil {
			t.Fatalf("executor error: %v", err)
		}
		if len(mockProvider.lastReq.Messages) == 0 {
			t.Fatalf("expected provider request messages")
		}
		return mockProvider.lastReq.Messages[0].Content
	}

	selectedPrompt := runTask(selectedAgent.ID)
	if !strings.Contains(selectedPrompt, selectedAgent.SystemPrompt) {
		t.Fatalf("expected selected agent prompt, got: %s", selectedPrompt)
	}

	defaultPrompt := runTask("")
	if !strings.Contains(defaultPrompt, defaultAgent.Name) {
		t.Fatalf("expected default agent name, got: %s", defaultPrompt)
	}
}

func TestResolveSubAgentID_PrefersEnabledSubagent(t *testing.T) {
	workspace := t.TempDir()
	store, err := storage.New(workspace, "", filepath.Join(workspace, "test.db"))
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	defer store.Close()

	masterAgent := &storage.Agent{
		Name:         "master-agent",
		Type:         storage.AgentTypeMaster,
		SystemPrompt: "master prompt",
		Enabled:      true,
	}
	if err := store.Agent().Save(masterAgent); err != nil {
		t.Fatalf("save master agent: %v", err)
	}
	masterAgent, err = store.Agent().GetByName("master-agent")
	if err != nil {
		t.Fatalf("get master agent: %v", err)
	}

	subAgent := &storage.Agent{
		Name:         "worker-agent",
		Type:         storage.AgentTypeSubAgent,
		SystemPrompt: "worker prompt",
		Enabled:      true,
	}
	if err := store.Agent().Save(subAgent); err != nil {
		t.Fatalf("save subagent: %v", err)
	}
	subAgent, err = store.Agent().GetByName("worker-agent")
	if err != nil {
		t.Fatalf("get subagent: %v", err)
	}

	resolvedID := resolveSubAgentID(store, masterAgent.ID)
	if resolvedID != subAgent.ID {
		t.Fatalf("resolved subagent id = %q, want %q", resolvedID, subAgent.ID)
	}
}

func writeWorkspaceFile(t *testing.T, workspace, name, content string) {
	t.Helper()
	path := filepath.Join(workspace, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func extractTaskID(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "Task ID: ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Task ID: "))
		}
	}
	return ""
}

func eventually(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}
