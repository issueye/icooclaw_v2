// Package scheduler provides task scheduling for icooclaw.
package scheduler

import (
	"context"
	"fmt"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/storage"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

const (
	TaskTypeImmediate = "immediate"
	TaskTypeScheduled = "scheduled"

	TaskExecutorMessage  = "message"
	TaskExecutorSummary  = "summary"
	TaskExecutorSubAgent = "subagent"

	TaskStatusPending = "pending"
	TaskStatusRunning = "running"
	TaskStatusSuccess = "success"
	TaskStatusFailed  = "failed"
)

// Task 定时任务.
type Task struct {
	ID          string       // 任务ID
	Name        string       // 任务名称
	TaskType    string       // 任务类型: immediate/scheduled
	Executor    string       // 执行器类型
	Schedule    string       // 任务调度表达式
	Description string       // 任务描述
	Content     string       // 任务内容 (消息文本)
	Params      string       // 任务参数 (JSON格式)
	Channel     string       // 任务通道名称
	SessionID   string       // 会话ID
	Enabled     bool         // 是否任务已启用
	LastRun     time.Time    // 上次运行时间
	NextRun     time.Time    // 下次运行时间
	LastStatus  string       // 最后执行状态
	LastError   string       // 最后一次执行错误
	Result      string       // 最近执行结果
	RunCount    int          // 累计执行次数
	EntryID     cron.EntryID // 任务条目ID
}

// TaskResult 任务执行结果。
type TaskResult struct {
	TaskID    string    // 任务ID
	StartTime time.Time // 开始时间
	EndTime   time.Time // 结束时间
	Result    string    // 结果文本
	Error     error     // 错误信息
}

// TaskExecutor 定义任务执行器。
type TaskExecutor func(ctx context.Context, task *Task) (string, error)

// Scheduler 定时任务调度器.
type Scheduler struct {
	cron    *cron.Cron
	tasks   map[string]*Task
	results chan TaskResult
	waiters map[string][]chan TaskResult
	logger  *slog.Logger
	mu      sync.RWMutex
	storage *storage.TaskStorage
	bus     *bus.MessageBus
	running bool
	execs   map[string]TaskExecutor
}

// NewScheduler 创建定时任务调度器.
func NewScheduler(storage *storage.TaskStorage, bus *bus.MessageBus, logger *slog.Logger) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Scheduler{
		cron:    cron.New(cron.WithLocation(time.Local)),
		tasks:   make(map[string]*Task),
		results: make(chan TaskResult, 100),
		waiters: make(map[string][]chan TaskResult),
		logger:  logger,
		storage: storage,
		bus:     bus,
		execs: map[string]TaskExecutor{
			TaskExecutorMessage: defaultMessageTaskExecutor(bus),
		},
	}
}

func defaultMessageTaskExecutor(messageBus *bus.MessageBus) TaskExecutor {
	return func(ctx context.Context, task *Task) (string, error) {
		// 构建消息文本: content + params
		messageText := task.Content
		if task.Params != "" {
			if messageText != "" {
				messageText += " " + task.Params
			} else {
				messageText = task.Params
			}
		}

		msg := bus.InboundMessage{
			Channel:   task.Channel,
			SessionID: task.SessionID,
			Text:      messageText,
			Timestamp: time.Now(),
			Metadata: map[string]any{
				"task_id":    task.ID,
				"task_name":  task.Name,
				"session_id": task.SessionID,
			},
		}
		if messageBus == nil {
			return "", fmt.Errorf("消息总线未初始化")
		}
		if err := messageBus.PublishInbound(ctx, msg); err != nil {
			return "", err
		}
		return "任务消息已投递到消息总线", nil
	}
}

// RegisterExecutor 注册任务执行器。
func (s *Scheduler) RegisterExecutor(name string, exec TaskExecutor) {
	if strings.TrimSpace(name) == "" || exec == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.execs[name] = exec
}

var scheduleAliases = map[string]string{
	"@every_minute": EveryMinute,
	"@every_5m":     Every5Minutes,
	"@every_15m":    Every15Minutes,
	"@every_30m":    Every30Minutes,
	"@every_hour":   EveryHour,
	"@every_2h":     Every2Hours,
	"@every_6h":     Every6Hours,
	"@every_12h":    Every12Hours,
	"@every_day":    EveryDay,
	"@every_week":   EveryWeek,
	"@every_month":  EveryMonth,
}

// NormalizeSchedule 将持续时间、别名或标准 cron 表达式归一化为标准 5 段 cron 表达式。
func NormalizeSchedule(expr string) (string, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", fmt.Errorf("调度表达式不能为空")
	}

	lowerExpr := strings.ToLower(expr)
	if normalized, ok := scheduleAliases[lowerExpr]; ok {
		return normalized, nil
	}

	if durationExpr, err := ParseDuration(expr); err == nil {
		return durationExpr, nil
	}

	if _, err := cron.ParseStandard(expr); err != nil {
		return "", fmt.Errorf("无效的调度表达式: %w", err)
	}

	return expr, nil
}

// ValidateSchedule 验证调度表达式是否合法。
func ValidateSchedule(expr string) error {
	_, err := NormalizeSchedule(expr)
	return err
}

func parseSchedule(expr string) (cron.Schedule, string, error) {
	normalized, err := NormalizeSchedule(expr)
	if err != nil {
		return nil, "", err
	}

	schedule, err := cron.ParseStandard(normalized)
	if err != nil {
		return nil, "", fmt.Errorf("无效的调度表达式: %w", err)
	}

	return schedule, normalized, nil
}

func nextRunForSchedule(expr string, from time.Time) (time.Time, error) {
	schedule, _, err := parseSchedule(expr)
	if err != nil {
		return time.Time{}, err
	}
	return schedule.Next(from), nil
}

func normalizeTaskType(taskType string) string {
	switch strings.ToLower(strings.TrimSpace(taskType)) {
	case "", TaskTypeScheduled:
		return TaskTypeScheduled
	case TaskTypeImmediate:
		return TaskTypeImmediate
	default:
		return strings.ToLower(strings.TrimSpace(taskType))
	}
}

func normalizeExecutor(executor string) string {
	executor = strings.ToLower(strings.TrimSpace(executor))
	if executor == "" {
		return TaskExecutorMessage
	}
	return executor
}

func (s *Scheduler) normalizeTask(task *Task) error {
	if task == nil {
		return fmt.Errorf("任务不能为空")
	}

	task.TaskType = normalizeTaskType(task.TaskType)
	task.Executor = normalizeExecutor(task.Executor)
	if task.TaskType != TaskTypeImmediate && task.TaskType != TaskTypeScheduled {
		return fmt.Errorf("不支持的任务类型: %s", task.TaskType)
	}

	if task.TaskType == TaskTypeScheduled {
		if strings.TrimSpace(task.Schedule) == "" {
			return fmt.Errorf("定时执行任务缺少 cron 表达式")
		}
		schedule, normalized, err := parseSchedule(task.Schedule)
		if err != nil {
			return err
		}
		_ = schedule
		task.Schedule = normalized
		return nil
	}

	task.Schedule = ""
	task.NextRun = time.Time{}
	return nil
}

// AddTask 添加定时任务。
func (s *Scheduler) AddTask(task *Task) error {
	return s.UpsertTask(task)
}

// UpsertTask 新增或替换定时任务。
func (s *Scheduler) UpsertTask(task *Task) error {
	s.mu.Lock()
	if err := s.upsertTaskLocked(task); err != nil {
		s.mu.Unlock()
		return err
	}

	snapshot := s.snapshotTaskLocked(s.tasks[task.ID])
	shouldDispatch := shouldDispatchImmediateTask(snapshot)
	s.mu.Unlock()
	if err := s.persistTaskState(snapshot); err != nil {
		return err
	}
	if shouldDispatch {
		go s.executeTaskByID(snapshot.ID)
	}
	return nil
}

// ApplyStorageTask 根据存储任务同步调度器状态。
func (s *Scheduler) ApplyStorageTask(task *storage.Task) (<-chan TaskResult, error) {
	if task == nil {
		return nil, fmt.Errorf("任务不能为空")
	}

	resultCh := s.registerTaskWaiter(task.ID)
	if err := s.UpsertTask(taskFromStorage(task)); err != nil {
		s.cancelTaskWaiter(task.ID, resultCh)
		return nil, err
	}
	return resultCh, nil
}

// DeleteTask 从调度器完全删除任务。
func (s *Scheduler) DeleteTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[id]
	if !exists {
		return nil
	}

	if task.EntryID != 0 {
		s.cron.Remove(task.EntryID)
	}
	delete(s.tasks, id)

	s.logger.Info("任务已从调度器删除", "id", id)
	return nil
}

func (s *Scheduler) addTaskLocked(task *Task) error {
	if err := s.normalizeTask(task); err != nil {
		return err
	}

	if task.TaskType == TaskTypeImmediate {
		task.EntryID = 0
		task.NextRun = time.Time{}
		if task.LastStatus == "" && task.Enabled {
			task.LastStatus = TaskStatusPending
		}
		s.tasks[task.ID] = task
		s.logger.Info("立即任务已添加", "id", task.ID, "name", task.Name, "executor", task.Executor)
		return nil
	}

	schedule, _, err := parseSchedule(task.Schedule)
	if err != nil {
		return err
	}

	entryID := s.cron.Schedule(schedule, cron.FuncJob(func() {
		s.executeTaskByID(task.ID)
	}))

	task.EntryID = entryID
	nextRun, err := nextRunForSchedule(task.Schedule, time.Now())
	if err != nil {
		return err
	}
	task.NextRun = nextRun
	if task.LastStatus == "" {
		task.LastStatus = TaskStatusPending
	}
	s.tasks[task.ID] = task

	s.logger.Info("定时任务已添加", "id", task.ID, "name", task.Name, "schedule", task.Schedule)
	return nil
}

func (s *Scheduler) upsertTaskLocked(task *Task) error {
	if task == nil {
		return fmt.Errorf("任务不能为空")
	}

	existing, exists := s.tasks[task.ID]
	if exists && existing.EntryID != 0 {
		s.cron.Remove(existing.EntryID)
	}

	if exists {
		if task.LastRun.IsZero() {
			task.LastRun = existing.LastRun
		}
		if task.LastStatus == "" {
			task.LastStatus = existing.LastStatus
		}
		if task.LastError == "" {
			task.LastError = existing.LastError
		}
		if task.RunCount == 0 {
			task.RunCount = existing.RunCount
		}
		if task.Result == "" {
			task.Result = existing.Result
		}
	}

	if task.Enabled {
		return s.addTaskLocked(task)
	}

	task.EntryID = 0
	task.NextRun = time.Time{}
	s.tasks[task.ID] = task
	s.logger.Info("任务已同步为禁用状态", "id", task.ID, "name", task.Name)
	return nil
}

// RemoveTask 删除定时任务.
func (s *Scheduler) RemoveTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[id]
	if !exists {
		return fmt.Errorf("任务ID %s 未找到", id)
	}

	if task.EntryID != 0 {
		s.cron.Remove(task.EntryID)
	}
	delete(s.tasks, id)

	s.logger.Info("任务已移除", "id", id)
	return nil
}

// EnableTask 启用定时任务.
func (s *Scheduler) EnableTask(id string) error {
	s.mu.Lock()
	task, exists := s.tasks[id]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("任务ID %s 未找到", id)
	}

	task.Enabled = true
	if err := s.upsertTaskLocked(task); err != nil {
		s.mu.Unlock()
		return err
	}
	snapshot := s.snapshotTaskLocked(s.tasks[id])
	s.mu.Unlock()
	if err := s.persistTaskState(snapshot); err != nil {
		return err
	}
	s.logger.Info("任务已启用", "id", id)
	return nil
}

// DisableTask 禁用定时任务.
func (s *Scheduler) DisableTask(id string) error {
	s.mu.Lock()
	task, exists := s.tasks[id]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("任务ID %s 未找到", id)
	}

	if task.EntryID != 0 {
		s.cron.Remove(task.EntryID)
	}
	task.Enabled = false
	task.EntryID = 0
	task.NextRun = time.Time{}
	snapshot := *task
	s.mu.Unlock()
	if err := s.persistTaskState(&snapshot); err != nil {
		return err
	}
	s.logger.Info("任务已禁用", "id", id)
	return nil
}

// GetTask 获取定时任务详情.
func (s *Scheduler) GetTask(id string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[id]
	if !exists {
		return nil, fmt.Errorf("任务ID %s 未找到", id)
	}

	return s.snapshotTaskLocked(task), nil
}

func shouldDispatchImmediateTask(task *Task) bool {
	if task == nil {
		return false
	}
	if task.TaskType != TaskTypeImmediate || !task.Enabled {
		return false
	}
	switch task.LastStatus {
	case "", TaskStatusPending, TaskStatusRunning:
		return true
	default:
		return false
	}
}

// ListTasks 列出所有定时任务.
func (s *Scheduler) ListTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, s.snapshotTaskLocked(task))
	}
	return tasks
}

// RunTask 立即执行定时任务.
func (s *Scheduler) RunTask(id string) (<-chan TaskResult, error) {
	resultCh := s.registerTaskWaiter(id)

	s.mu.Lock()
	task, exists := s.tasks[id]
	if !exists {
		s.mu.Unlock()
		s.cancelTaskWaiter(id, resultCh)
		return nil, fmt.Errorf("任务ID %s 未找到", id)
	}

	if task.TaskType == TaskTypeImmediate {
		task.Enabled = true
		task.LastStatus = TaskStatusPending
		task.LastError = ""
		task.Result = ""
		snapshot := s.snapshotTaskLocked(task)
		s.mu.Unlock()
		if err := s.persistTaskState(snapshot); err != nil {
			s.cancelTaskWaiter(id, resultCh)
			return nil, err
		}
		go s.executeTaskByID(id)
		return resultCh, nil
	}

	if !task.Enabled {
		s.mu.Unlock()
		s.cancelTaskWaiter(id, resultCh)
		return nil, fmt.Errorf("任务ID %s 当前处于禁用状态", id)
	}
	s.mu.Unlock()

	go s.executeTaskByID(id)
	return resultCh, nil
}

func (s *Scheduler) LoadTasks() error {
	s.mu.Lock()
	s.tasks = make(map[string]*Task)
	s.mu.Unlock()

	tasks, err := s.storage.GetAll()
	if err != nil {
		return fmt.Errorf("加载任务失败: %w", err)
	}

	for _, task := range tasks {
		t := taskFromStorage(&task)
		if err := s.UpsertTask(t); err != nil {
			s.logger.Warn("加载任务到调度器失败", "id", task.ID, "name", task.Name, "error", err)
			continue
		}
	}
	return nil
}

// Start 启动定时任务调度器.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	s.cron.Start()

	s.running = true
	s.logger.Info("调度器已启动")
}

// Stop 停止定时任务调度器.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	ctx := s.cron.Stop()
	<-ctx.Done()
	s.running = false
	s.logger.Info("调度器已停止")
}

// Results 返回任务执行结果通道.
func (s *Scheduler) Results() <-chan TaskResult {
	return s.results
}

// IsRunning 是否正在运行.
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// executeTask 执行定时任务.
func (s *Scheduler) executeTaskByID(id string) {
	s.mu.RLock()
	task, exists := s.tasks[id]
	if !exists {
		s.mu.RUnlock()
		return
	}
	snapshot := s.snapshotTaskLocked(task)
	s.mu.RUnlock()

	s.executeTask(snapshot)
}

func (s *Scheduler) executeTask(task *Task) {
	startTime := time.Now()
	s.logger.Debug("正在执行任务", "id", task.ID, "name", task.Name)

	s.markTaskRunning(task.ID)

	s.logger.Info("执行任务", "task_id", task.ID, "task_name", task.Name, "task_type", task.TaskType, "executor", task.Executor)

	execResult, execErr := s.runTaskExecutor(task)

	endTime := time.Now()
	s.updateTaskExecutionState(task.ID, endTime, execResult, execErr)

	// Send result to channel
	result := TaskResult{
		TaskID:    task.ID,
		StartTime: startTime,
		EndTime:   endTime,
		Result:    execResult,
		Error:     execErr,
	}

	select {
	case s.results <- result:
	default:
		s.logger.Warn("结果通道已满，丢弃结果", "task_id", task.ID)
	}

	s.publishTaskResult(result)
}

func (s *Scheduler) runTaskExecutor(task *Task) (string, error) {
	s.mu.RLock()
	exec := s.execs[normalizeExecutor(task.Executor)]
	s.mu.RUnlock()
	if exec == nil {
		return "", fmt.Errorf("未找到任务执行器: %s", task.Executor)
	}

	result, err := exec(context.Background(), task)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

func (s *Scheduler) snapshotTaskLocked(task *Task) *Task {
	if task == nil {
		return nil
	}

	snapshot := *task
	if task.EntryID != 0 {
		if entry := s.cron.Entry(task.EntryID); entry.ID != 0 {
			snapshot.NextRun = entry.Next
		}
	}
	if snapshot.TaskType == TaskTypeScheduled && snapshot.Enabled && snapshot.NextRun.IsZero() {
		nextRun, err := nextRunForSchedule(snapshot.Schedule, time.Now())
		if err == nil {
			snapshot.NextRun = nextRun
		}
	}

	return &snapshot
}

func (s *Scheduler) markTaskRunning(id string) {
	s.mu.Lock()
	task, exists := s.tasks[id]
	if !exists {
		s.mu.Unlock()
		return
	}

	task.LastStatus = TaskStatusRunning
	task.LastError = ""
	task.Result = ""
	snapshot := *task
	s.mu.Unlock()

	if err := s.persistTaskState(&snapshot); err != nil {
		s.logger.Warn("回写任务运行中状态失败", "id", id, "error", err)
	}
}

func (s *Scheduler) updateTaskExecutionState(id string, finishedAt time.Time, execResult string, execErr error) {
	s.mu.Lock()
	task, exists := s.tasks[id]
	if !exists {
		s.mu.Unlock()
		return
	}

	task.LastRun = finishedAt
	task.RunCount++
	if execErr != nil {
		task.LastStatus = TaskStatusFailed
		task.LastError = execErr.Error()
		task.Result = ""
	} else {
		task.LastStatus = TaskStatusSuccess
		task.LastError = ""
		task.Result = execResult
	}

	if task.TaskType == TaskTypeImmediate {
		task.Enabled = false
		task.EntryID = 0
		task.NextRun = time.Time{}
	} else if task.EntryID != 0 {
		if entry := s.cron.Entry(task.EntryID); entry.ID != 0 {
			task.NextRun = entry.Next
		}
	}
	if task.TaskType == TaskTypeScheduled && task.Enabled && task.NextRun.IsZero() {
		nextRun, err := nextRunForSchedule(task.Schedule, finishedAt)
		if err == nil {
			task.NextRun = nextRun
		}
	}
	if !task.Enabled {
		task.NextRun = time.Time{}
	}

	snapshot := *task
	s.mu.Unlock()

	if err := s.persistTaskState(&snapshot); err != nil {
		s.logger.Warn("回写任务运行状态失败", "id", id, "error", err)
	}
}

func (s *Scheduler) persistTaskState(task *Task) error {
	if s.storage == nil || task == nil {
		return nil
	}

	return s.storage.UpdateRuntimeState(task.ID, storage.TaskRuntimeState{
		LastRunAt:  formatTaskTime(task.LastRun),
		NextRunAt:  formatTaskTime(task.NextRun),
		LastStatus: task.LastStatus,
		LastError:  task.LastError,
		Result:     task.Result,
		Enabled:    task.Enabled,
		RunCount:   task.RunCount,
	})
}

func (s *Scheduler) registerTaskWaiter(taskID string) chan TaskResult {
	ch := make(chan TaskResult, 1)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.waiters[taskID] = append(s.waiters[taskID], ch)
	return ch
}

func (s *Scheduler) cancelTaskWaiter(taskID string, target chan TaskResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	waiters := s.waiters[taskID]
	if len(waiters) == 0 {
		return
	}

	filtered := waiters[:0]
	for _, ch := range waiters {
		if ch == target {
			close(ch)
			continue
		}
		filtered = append(filtered, ch)
	}

	if len(filtered) == 0 {
		delete(s.waiters, taskID)
		return
	}
	s.waiters[taskID] = filtered
}

func (s *Scheduler) publishTaskResult(result TaskResult) {
	s.mu.Lock()
	waiters := s.waiters[result.TaskID]
	if len(waiters) > 0 {
		delete(s.waiters, result.TaskID)
	}
	s.mu.Unlock()

	for _, ch := range waiters {
		ch <- result
		close(ch)
	}
}

func taskFromStorage(task *storage.Task) *Task {
	if task == nil {
		return nil
	}

	return &Task{
		ID:          task.ID,
		Name:        task.Name,
		TaskType:    task.TaskType,
		Executor:    task.Executor,
		Schedule:    task.CronExpr,
		Description: task.Description,
		Content:     task.Content,
		Params:      task.Params,
		Channel:     task.Channel,
		SessionID:   task.SessionID,
		Enabled:     task.Enabled,
		LastRun:     parseTaskTime(task.LastRunAt),
		NextRun:     parseTaskTime(task.NextRunAt),
		LastStatus:  task.LastStatus,
		LastError:   task.LastError,
		Result:      task.Result,
		RunCount:    task.RunCount,
	}
}

func formatTaskTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func parseTaskTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed
		}
	}

	return time.Time{}
}

// Common 定时任务调度.
const (
	EveryMinute    = "* * * * *"    // 每分钟执行一次
	Every5Minutes  = "*/5 * * * *"  // 每5分钟执行一次
	Every15Minutes = "*/15 * * * *" // 每15分钟执行一次
	Every30Minutes = "*/30 * * * *" // 每30分钟执行一次
	EveryHour      = "0 * * * *"    // 每小时执行一次
	Every2Hours    = "0 */2 * * *"  // 每2小时执行一次
	Every6Hours    = "0 */6 * * *"  // 每6小时执行一次
	Every12Hours   = "0 */12 * * *" // 每12小时执行一次
	EveryDay       = "0 0 * * *"    // 每天执行一次
	EveryWeek      = "0 0 * * 0"    // 每周执行一次（周日）
	EveryMonth     = "0 0 1 * *"    // 每月1号执行一次
)

// ParseDuration 解析持续时间字符串并返回定时任务调度表达式.
func ParseDuration(d string) (string, error) {
	duration, err := time.ParseDuration(d)
	if err != nil {
		return "", err
	}

	// Convert to cron schedule
	switch {
	case duration < time.Minute:
		return "", fmt.Errorf("最小持续时间为 1 分钟")
	case duration == time.Minute:
		return EveryMinute, nil
	case duration%time.Hour == 0:
		hours := int(duration / time.Hour)
		if hours == 1 {
			return EveryHour, nil
		}
		return fmt.Sprintf("0 */%d * * *", hours), nil
	case duration%time.Minute == 0:
		minutes := int(duration / time.Minute)
		if minutes == 1 {
			return EveryMinute, nil
		}
		return fmt.Sprintf("*/%d * * * *", minutes), nil
	default:
		return "", fmt.Errorf("不支持的持续时间: %s", d)
	}
}
