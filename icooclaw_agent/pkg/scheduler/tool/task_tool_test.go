package tool

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/scheduler"
	"icooclaw/pkg/storage"
)

func TestTaskCreateToolImmediateTaskWithoutCron(t *testing.T) {
	store := newTaskToolStorage(t)
	messageBus := bus.NewMessageBus(bus.DefaultConfig())
	taskScheduler := scheduler.NewScheduler(store.Task(), messageBus, nil)
	tool := NewTaskCreateTool(store.Task(), taskScheduler, messageBus, nil)

	result := tool.Execute(context.Background(), map[string]any{
		"name":        "instant-task",
		"description": "instant task",
		"task_type":   scheduler.TaskTypeImmediate,
		"content":     "hello",
		"channel":     "web",
		"session_id":  "session-1",
	})
	if result.Error != nil {
		t.Fatalf("task create failed: %v", result.Error)
	}

	taskID := extractCreatedTaskID(result.Content)
	if taskID == "" {
		t.Fatalf("expected task id in result: %s", result.Content)
	}

	select {
	case execResult := <-taskScheduler.Results():
		if execResult.Error != nil {
			t.Fatalf("task execution failed: %v", execResult.Error)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiting immediate task execution timed out")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	msg, ok := messageBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message for immediate task")
	}
	if msg.Text != "hello" {
		t.Fatalf("message text = %q, want %q", msg.Text, "hello")
	}

	task, err := store.Task().GetByID(taskID)
	if err != nil {
		t.Fatalf("load task: %v", err)
	}
	if task.TaskType != scheduler.TaskTypeImmediate {
		t.Fatalf("task type = %q, want %q", task.TaskType, scheduler.TaskTypeImmediate)
	}
	if task.CronExpr != "" {
		t.Fatalf("cron expr = %q, want empty", task.CronExpr)
	}
	if task.LastStatus != scheduler.TaskStatusSuccess {
		t.Fatalf("last status = %q, want %q", task.LastStatus, scheduler.TaskStatusSuccess)
	}
	if task.Enabled {
		t.Fatal("immediate task should be disabled after execution")
	}
}

func TestTaskCreateToolScheduledRequiresCron(t *testing.T) {
	store := newTaskToolStorage(t)
	tool := NewTaskCreateTool(store.Task(), scheduler.NewScheduler(store.Task(), nil, nil), nil, nil)

	result := tool.Execute(context.Background(), map[string]any{
		"name":        "scheduled-task",
		"description": "scheduled task",
		"task_type":   scheduler.TaskTypeScheduled,
		"content":     "hello",
		"channel":     "web",
		"session_id":  "session-1",
	})
	if result.Error == nil {
		t.Fatal("expected error when scheduled task misses cron_expr")
	}
}

func TestTaskUpdateToolSwitchToImmediateClearsCron(t *testing.T) {
	store := newTaskToolStorage(t)
	task := &storage.Task{
		Name:        "scheduled-task",
		Description: "scheduled task",
		TaskType:    scheduler.TaskTypeScheduled,
		Executor:    scheduler.TaskExecutorMessage,
		Content:     "hello",
		Channel:     "web",
		SessionID:   "session-1",
		CronExpr:    scheduler.EveryHour,
		Enabled:     false,
	}
	if err := store.Task().Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	tool := NewTaskUpdateTool(store.Task(), scheduler.NewScheduler(store.Task(), nil, nil), nil, nil)
	result := tool.Execute(context.Background(), map[string]any{
		"task_id":   task.ID,
		"task_type": scheduler.TaskTypeImmediate,
	})
	if result.Error != nil {
		t.Fatalf("task update failed: %v", result.Error)
	}

	updated, err := store.Task().GetByID(task.ID)
	if err != nil {
		t.Fatalf("load updated task: %v", err)
	}
	if updated.TaskType != scheduler.TaskTypeImmediate {
		t.Fatalf("task type = %q, want %q", updated.TaskType, scheduler.TaskTypeImmediate)
	}
	if updated.CronExpr != "" {
		t.Fatalf("cron expr = %q, want empty", updated.CronExpr)
	}
}

func newTaskToolStorage(t *testing.T) *storage.Storage {
	t.Helper()
	workspace := t.TempDir()
	store, err := storage.New(workspace, "", filepath.Join(workspace, "test.db"))
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}

func extractCreatedTaskID(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "ID: ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "ID: "))
		}
	}
	return ""
}
