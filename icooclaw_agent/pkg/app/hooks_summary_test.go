package app

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"icooclaw/pkg/agent/react"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/scheduler"
	"icooclaw/pkg/storage"
)

func TestMaybeSummarizeSessionQueuesAndExecutesTask(t *testing.T) {
	workspace := t.TempDir()
	store, err := storage.New(workspace, "", filepath.Join(workspace, "test.db"))
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	defer store.Close()

	if err := store.Param().Save(&storage.ParamConfig{
		Key:   consts.DEFAULT_MODEL_KEY,
		Value: "mock/mock-summary",
	}); err != nil {
		t.Fatalf("save default model: %v", err)
	}

	manager := providers.NewManager(store, nil)
	manager.Register("mock", &mockSummaryProvider{response: "这是新的摘要"})

	summaryAgent, err := react.NewReActAgentNoHooks(context.Background(), react.Dependencies{
		ProviderManager: manager,
		Storage:         store,
		Logger:          testHooksLogger(),
	})
	if err != nil {
		t.Fatalf("create summary agent: %v", err)
	}

	taskScheduler := scheduler.NewScheduler(store.Task(), nil, testHooksLogger())
	hooks := NewAppAgentHooks(HooksDependencies{
		Logger:       testHooksLogger(),
		Workspace:    workspace,
		Storage:      store,
		Scheduler:    taskScheduler,
		SummaryAgent: summaryAgent,
		RecentCount:  2,
	})
	taskScheduler.RegisterExecutor(scheduler.TaskExecutorSummary, hooks.ExecuteSummaryTask)

	sessionKey := consts.GetSessionKey(consts.WEB, "session-1")
	saveHookMessage(t, store, sessionKey, consts.RoleUser, "你好")
	saveHookMessage(t, store, sessionKey, consts.RoleAssistant, "你好，我在")
	saveHookMessage(t, store, sessionKey, consts.RoleUser, "请整理一下")
	saveHookMessage(t, store, sessionKey, consts.RoleAssistant, "好的")

	summary, err := hooks.maybeSummarizeSession(context.Background(), bus.InboundMessage{
		Channel:   consts.WEB,
		SessionID: "session-1",
	}, sessionKey, "final response", 3)
	if err != nil {
		t.Fatalf("maybeSummarizeSession: %v", err)
	}
	if summary != "" {
		t.Fatalf("summary should be queued asynchronously, got %q", summary)
	}

	waitUntil(t, 2*time.Second, func() bool {
		tasks, err := store.Task().GetAll()
		if err != nil || len(tasks) != 1 {
			return false
		}
		return tasks[0].LastStatus == scheduler.TaskStatusSuccess
	})

	tasks, err := store.Task().GetAll()
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 summary task, got %d", len(tasks))
	}
	if tasks[0].TaskType != scheduler.TaskTypeImmediate {
		t.Fatalf("task type = %q, want %q", tasks[0].TaskType, scheduler.TaskTypeImmediate)
	}
	if tasks[0].Executor != scheduler.TaskExecutorSummary {
		t.Fatalf("executor = %q, want %q", tasks[0].Executor, scheduler.TaskExecutorSummary)
	}
	if tasks[0].Result != "这是新的摘要" {
		t.Fatalf("task result = %q, want %q", tasks[0].Result, "这是新的摘要")
	}

	summaries, err := store.Message().GetSummary(sessionKey)
	if err != nil {
		t.Fatalf("get summary messages: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary message, got %d", len(summaries))
	}
	if summaries[0].Content != "这是新的摘要" {
		t.Fatalf("summary content = %q, want %q", summaries[0].Content, "这是新的摘要")
	}
}

func TestMaybeSummarizeSessionSkipsWhenRecentMessagesNotEnough(t *testing.T) {
	workspace := t.TempDir()
	store, err := storage.New(workspace, "", filepath.Join(workspace, "test.db"))
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	defer store.Close()

	hooks := NewAppAgentHooks(HooksDependencies{
		Logger:      testHooksLogger(),
		Workspace:   workspace,
		Storage:     store,
		Scheduler:   scheduler.NewScheduler(store.Task(), nil, testHooksLogger()),
		RecentCount: 2,
	})

	sessionKey := consts.GetSessionKey(consts.WEB, "session-2")
	saveHookMessage(t, store, sessionKey, consts.RoleUser, "hello")
	saveHookMessage(t, store, sessionKey, consts.RoleAssistant, "world")

	summary, err := hooks.maybeSummarizeSession(context.Background(), bus.InboundMessage{
		Channel:   consts.WEB,
		SessionID: "session-2",
	}, sessionKey, "final response", 1)
	if err != nil {
		t.Fatalf("maybeSummarizeSession: %v", err)
	}
	if summary != "" {
		t.Fatalf("expected empty summary result, got %q", summary)
	}

	tasks, err := store.Task().GetAll()
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected 0 queued tasks, got %d", len(tasks))
	}
}

func saveHookMessage(t *testing.T, store *storage.Storage, sessionKey string, role consts.RoleType, content string) {
	t.Helper()
	if err := store.Message().Save(&storage.Message{
		SessionID: sessionKey,
		Role:      role,
		Content:   content,
	}); err != nil {
		t.Fatalf("save message: %v", err)
	}
}

type mockSummaryProvider struct {
	response string
}

func (m *mockSummaryProvider) Chat(ctx context.Context, req providers.ChatRequest) (*providers.ChatResponse, error) {
	return &providers.ChatResponse{Content: m.response}, nil
}

func (m *mockSummaryProvider) ChatStream(ctx context.Context, req providers.ChatRequest, callback providers.StreamCallback) error {
	return callback(m.response, "", nil, true)
}

func (m *mockSummaryProvider) GetName() string {
	return "mock"
}

func (m *mockSummaryProvider) GetModel() string {
	return "mock-summary"
}

func (m *mockSummaryProvider) SetModel(model string) {}

func waitUntil(t *testing.T, timeout time.Duration, fn func() bool) {
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
