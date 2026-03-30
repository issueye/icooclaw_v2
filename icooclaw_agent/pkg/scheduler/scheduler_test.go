package scheduler

import (
	"context"
	"testing"
	"time"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/storage"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{"1 minute", "1m", EveryMinute, false},
		{"5 minutes", "5m", Every5Minutes, false},
		{"15 minutes", "15m", Every15Minutes, false},
		{"30 minutes", "30m", Every30Minutes, false},
		{"1 hour", "1h", EveryHour, false},
		{"2 hours", "2h", "0 */2 * * *", false},
		{"6 hours", "6h", "0 */6 * * *", false},
		{"12 hours", "12h", "0 */12 * * *", false},
		{"1 second", "1s", "", true},
		{"30 seconds", "30s", "", true},
		{"90 minutes", "90m", "*/90 * * * *", false},
		{"3 hours", "3h", "0 */3 * * *", false},
		{"invalid", "invalid", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("ParseDuration(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTask_Structure(t *testing.T) {
	task := &Task{
		ID:          "test-id",
		Name:        "Test Task",
		Schedule:    EveryMinute,
		Description: "Test description",
		Content:     "Hello World",
		Params:      `{"key":"value"}`,
		Channel:     "qq",
		SessionID:   "session-123",
		Enabled:     true,
		LastRun:     time.Now(),
		NextRun:     time.Now().Add(time.Minute),
		EntryID:     1,
	}

	if task.ID != "test-id" {
		t.Errorf("ID = %q, want %q", task.ID, "test-id")
	}
	if task.Name != "Test Task" {
		t.Errorf("Name = %q, want %q", task.Name, "Test Task")
	}
	if task.Schedule != EveryMinute {
		t.Errorf("Schedule = %q, want %q", task.Schedule, EveryMinute)
	}
	if task.Content != "Hello World" {
		t.Errorf("Content = %q, want %q", task.Content, "Hello World")
	}
	if task.Params != `{"key":"value"}` {
		t.Errorf("Params = %q, want %q", task.Params, `{"key":"value"}`)
	}
	if task.Channel != "qq" {
		t.Errorf("Channel = %q, want %q", task.Channel, "qq")
	}
	if task.SessionID != "session-123" {
		t.Errorf("SessionID = %q, want %q", task.SessionID, "session-123")
	}
	if !task.Enabled {
		t.Error("Enabled should be true")
	}
}

func TestTaskResult_Structure(t *testing.T) {
	start := time.Now()
	end := start.Add(time.Second)
	result := &TaskResult{
		TaskID:    "task-123",
		StartTime: start,
		EndTime:   end,
		Error:     nil,
	}

	if result.TaskID != "task-123" {
		t.Errorf("TaskID = %q, want %q", result.TaskID, "task-123")
	}
	if !result.StartTime.Equal(start) {
		t.Errorf("StartTime = %v, want %v", result.StartTime, start)
	}
	if !result.EndTime.Equal(end) {
		t.Errorf("EndTime = %v, want %v", result.EndTime, end)
	}
	if result.Error != nil {
		t.Error("Error should be nil")
	}
}

func TestSchedulerConstants(t *testing.T) {
	if EveryMinute != "* * * * *" {
		t.Errorf("EveryMinute = %q, want %q", EveryMinute, "* * * * *")
	}
	if Every5Minutes != "*/5 * * * *" {
		t.Errorf("Every5Minutes = %q, want %q", Every5Minutes, "*/5 * * * *")
	}
	if Every15Minutes != "*/15 * * * *" {
		t.Errorf("Every15Minutes = %q, want %q", Every15Minutes, "*/15 * * * *")
	}
	if Every30Minutes != "*/30 * * * *" {
		t.Errorf("Every30Minutes = %q, want %q", Every30Minutes, "*/30 * * * *")
	}
	if EveryHour != "0 * * * *" {
		t.Errorf("EveryHour = %q, want %q", EveryHour, "0 * * * *")
	}
	if Every2Hours != "0 */2 * * *" {
		t.Errorf("Every2Hours = %q, want %q", Every2Hours, "0 */2 * * *")
	}
	if Every6Hours != "0 */6 * * *" {
		t.Errorf("Every6Hours = %q, want %q", Every6Hours, "0 */6 * * *")
	}
	if Every12Hours != "0 */12 * * *" {
		t.Errorf("Every12Hours = %q, want %q", Every12Hours, "0 */12 * * *")
	}
	if EveryDay != "0 0 * * *" {
		t.Errorf("EveryDay = %q, want %q", EveryDay, "0 0 * * *")
	}
	if EveryWeek != "0 0 * * 0" {
		t.Errorf("EveryWeek = %q, want %q", EveryWeek, "0 0 * * 0")
	}
	if EveryMonth != "0 0 1 * *" {
		t.Errorf("EveryMonth = %q, want %q", EveryMonth, "0 0 1 * *")
	}
}

func TestNormalizeSchedule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{"alias", "@every_day", EveryDay, false},
		{"duration", "2h", Every2Hours, false},
		{"standard", "0 9 * * 1-5", "0 9 * * 1-5", false},
		{"invalid", "bad cron", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeSchedule(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizeSchedule(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.expected {
				t.Fatalf("NormalizeSchedule(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestScheduler_LoadTasksAndPersistRuntimeState(t *testing.T) {
	store := newTestTaskStorage(t)
	task := &storage.Task{
		Name:        "daily report",
		Description: "send report",
		Content:     "report",
		Channel:     "local",
		SessionID:   "session-1",
		CronExpr:    "@every_hour",
		Enabled:     true,
	}
	if err := store.Task().Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	s := NewScheduler(store.Task(), bus.NewMessageBus(bus.DefaultConfig()), nil)
	if err := s.LoadTasks(); err != nil {
		t.Fatalf("LoadTasks() error = %v", err)
	}

	loaded, err := s.GetTask(task.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if loaded.Schedule != EveryHour {
		t.Fatalf("loaded schedule = %q, want %q", loaded.Schedule, EveryHour)
	}
	if loaded.NextRun.IsZero() {
		t.Fatal("loaded next run should not be zero")
	}

	persisted, err := store.Task().GetByID(task.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if persisted.NextRunAt == "" {
		t.Fatal("persisted next_run_at should not be empty after LoadTasks")
	}
}

func TestScheduler_RunTaskPersistsExecutionState(t *testing.T) {
	store := newTestTaskStorage(t)
	task := &storage.Task{
		Name:        "manual report",
		Description: "send report",
		Content:     "hello",
		Channel:     "local",
		SessionID:   "session-1",
		CronExpr:    EveryMinute,
		Enabled:     true,
	}
	if err := store.Task().Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	messageBus := bus.NewMessageBus(bus.DefaultConfig())
	s := NewScheduler(store.Task(), messageBus, nil)
	if err := s.LoadTasks(); err != nil {
		t.Fatalf("LoadTasks() error = %v", err)
	}

	resultCh, err := s.RunTask(task.ID)
	if err != nil {
		t.Fatalf("RunTask() error = %v", err)
	}

	select {
	case result := <-resultCh:
		if result.Error != nil {
			t.Fatalf("task execution returned error: %v", result.Error)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiting task result timed out")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	msg, ok := messageBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message from task execution")
	}
	if msg.Text != "hello" {
		t.Fatalf("inbound text = %q, want %q", msg.Text, "hello")
	}

	persisted, err := store.Task().GetByID(task.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if persisted.LastStatus != TaskStatusSuccess {
		t.Fatalf("last_status = %q, want %q", persisted.LastStatus, TaskStatusSuccess)
	}
	if persisted.LastRunAt == "" {
		t.Fatal("last_run_at should not be empty")
	}
	if persisted.RunCount != 1 {
		t.Fatalf("run_count = %d, want 1", persisted.RunCount)
	}
}

func TestScheduler_RunTaskRejectsDisabledTask(t *testing.T) {
	s := NewScheduler(nil, nil, nil)
	if err := s.UpsertTask(&Task{ID: "task-1", Name: "disabled", TaskType: TaskTypeScheduled, Schedule: EveryHour, Enabled: false}); err != nil {
		t.Fatalf("UpsertTask() error = %v", err)
	}

	if _, err := s.RunTask("task-1"); err == nil {
		t.Fatal("RunTask() should reject disabled task")
	}
}

func TestScheduler_ImmediateTaskExecutesAndStoresResult(t *testing.T) {
	store := newTestTaskStorage(t)
	s := NewScheduler(store.Task(), nil, nil)
	s.RegisterExecutor("echo", func(ctx context.Context, task *Task) (string, error) {
		return "echo:" + task.Content, nil
	})

	task := &storage.Task{
		Name:       "instant",
		TaskType:   TaskTypeImmediate,
		Executor:   "echo",
		Content:    "hello",
		Enabled:    true,
		LastStatus: TaskStatusPending,
	}
	if err := store.Task().Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	resultCh, err := s.ApplyStorageTask(task)
	if err != nil {
		t.Fatalf("ApplyStorageTask() error = %v", err)
	}

	select {
	case result := <-resultCh:
		if result.Error != nil {
			t.Fatalf("task execution returned error: %v", result.Error)
		}
		if result.Result != "echo:hello" {
			t.Fatalf("task result = %q, want %q", result.Result, "echo:hello")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiting immediate task result timed out")
	}

	persisted, err := store.Task().GetByID(task.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if persisted.LastStatus != TaskStatusSuccess {
		t.Fatalf("last_status = %q, want %q", persisted.LastStatus, TaskStatusSuccess)
	}
	if persisted.Result != "echo:hello" {
		t.Fatalf("result = %q, want %q", persisted.Result, "echo:hello")
	}
	if persisted.Enabled {
		t.Fatal("immediate task should be disabled after completion")
	}
}

func newTestTaskStorage(t *testing.T) *storage.Storage {
	t.Helper()

	dir := t.TempDir()
	store, err := storage.New(dir, "", dir+"\\test.db")
	if err != nil {
		t.Fatalf("storage.New() error = %v", err)
	}

	t.Cleanup(func() {
		_ = store.Close()
	})

	return store
}
