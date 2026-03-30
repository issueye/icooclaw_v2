package tool

import (
	"context"
	"fmt"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/scheduler"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
	"log/slog"
)

// TaskEnableTool 定时任务启用工具
type TaskEnableTool struct {
	scheduler *scheduler.Scheduler
	store     *storage.TaskStorage
	logger    *slog.Logger
	bus       *bus.MessageBus
}

// NewTaskEnableTool 创建启用定时任务工具
func NewTaskEnableTool(store *storage.TaskStorage, scheduler *scheduler.Scheduler, bus *bus.MessageBus, logger *slog.Logger) *TaskEnableTool {
	if logger == nil {
		logger = slog.Default()
	}
	return &TaskEnableTool{
		store:     store,
		scheduler: scheduler,
		logger:    logger,
		bus:       bus,
	}
}

// Name 工具名称
func (t *TaskEnableTool) Name() string {
	return "task_enable"
}

// Description 工具描述
func (t *TaskEnableTool) Description() string {
	return "启用一个被禁用的定时任务，使其恢复正常调度执行。\n\n效果:\n- 任务将按照 cron 表达式自动调度\n- 下次执行时间根据 cron 表达式计算\n- 任务状态变为\"已启用\""
}

// Parameters 工具参数
func (t *TaskEnableTool) Parameters() map[string]any {
	return map[string]any{
		"task_id": map[string]any{
			"type":        "string",
			"description": "要启用的任务 ID",
		},
		"required": []string{"task_id"},
	}
}

// Execute 执行启用任务
func (t *TaskEnableTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return tools.ErrorResult("需要提供 task_id 参数")
	}

	task, err := t.store.GetByID(taskID)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("获取任务失败: %v", err))
	}

	if task.Enabled {
		return tools.SuccessResult(fmt.Sprintf("任务 %s 已经是启用状态", taskID))
	}

	task.Enabled = true
	if err := t.store.Update(task); err != nil {
		return tools.ErrorResult(fmt.Sprintf("启用任务失败: %v", err))
	}

	if err := syncSchedulerTask(t.scheduler, task); err != nil {
		return tools.ErrorResult(fmt.Sprintf("同步调度器失败: %v", err))
	}

	return tools.SuccessResult(fmt.Sprintf("任务已启用\n\n任务ID: %s\n名称: %s\n调度: %s", taskID, task.Name, task.CronExpr))
}
