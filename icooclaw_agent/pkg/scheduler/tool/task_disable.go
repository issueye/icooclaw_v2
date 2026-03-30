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

// TaskDisableTool 定时任务禁用工具
type TaskDisableTool struct {
	scheduler *scheduler.Scheduler
	store     *storage.TaskStorage
	logger    *slog.Logger
	bus       *bus.MessageBus
}

// NewTaskDisableTool 创建禁用定时任务工具
func NewTaskDisableTool(store *storage.TaskStorage, scheduler *scheduler.Scheduler, bus *bus.MessageBus, logger *slog.Logger) *TaskDisableTool {
	if logger == nil {
		logger = slog.Default()
	}
	return &TaskDisableTool{
		store:     store,
		scheduler: scheduler,
		logger:    logger,
		bus:       bus,
	}
}

// Name 工具名称
func (t *TaskDisableTool) Name() string {
	return "task_disable"
}

// Description 工具描述
func (t *TaskDisableTool) Description() string {
	return "禁用一个已启用的定时任务，暂停其调度执行。\n\n效果:\n- 任务从调度器中移除，不再自动执行\n- 任务数据和配置保留不变\n- 任务状态变为\"已禁用\"\n- 随时可以使用 task_enable 重新启用"
}

// Parameters 工具参数
func (t *TaskDisableTool) Parameters() map[string]any {
	return map[string]any{
		"task_id": map[string]any{
			"type":        "string",
			"description": "要禁用的任务 ID",
		},
		"required": []string{"task_id"},
	}
}

// Execute 执行禁用任务
func (t *TaskDisableTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return tools.ErrorResult("需要提供 task_id 参数")
	}

	task, err := t.store.GetByID(taskID)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("获取任务失败: %v", err))
	}

	if !task.Enabled {
		return tools.SuccessResult(fmt.Sprintf("任务 %s 已经是禁用状态", taskID))
	}

	task.Enabled = false
	if err := t.store.Update(task); err != nil {
		return tools.ErrorResult(fmt.Sprintf("禁用任务失败: %v", err))
	}

	if err := syncSchedulerTask(t.scheduler, task); err != nil {
		return tools.ErrorResult(fmt.Sprintf("同步调度器失败: %v", err))
	}

	return tools.SuccessResult(fmt.Sprintf("任务已禁用\n\n任务ID: %s\n名称: %s\n\n任务数据已保留，随时可以重新启用", taskID, task.Name))
}
