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

// TaskDeleteTool 定时任务删除工具
type TaskDeleteTool struct {
	scheduler *scheduler.Scheduler
	store     *storage.TaskStorage
	logger    *slog.Logger
	bus       *bus.MessageBus
}

// NewTaskDeleteTool 创建定时任务删除工具
func NewTaskDeleteTool(store *storage.TaskStorage, scheduler *scheduler.Scheduler, bus *bus.MessageBus, logger *slog.Logger) *TaskDeleteTool {
	if logger == nil {
		logger = slog.Default()
	}
	return &TaskDeleteTool{
		store:     store,
		scheduler: scheduler,
		logger:    logger,
		bus:       bus,
	}
}

// Name 工具名称
func (t *TaskDeleteTool) Name() string {
	return "task_delete"
}

// Description 工具描述
func (t *TaskDeleteTool) Description() string {
	return "删除指定的定时任务。\n\n重要提示:\n- 删除操作不可恢复\n- 如果任务正在运行，会等当前执行完成后才删除\n- 任务的所有历史记录也会被删除\n\n建议在删除前先使用 task_get 确认任务信息"
}

// Parameters 工具参数
func (t *TaskDeleteTool) Parameters() map[string]any {
	return map[string]any{
		"task_id": map[string]any{
			"type":        "string",
			"description": "要删除的任务 ID",
		},
		"required": []string{"task_id"},
	}
}

func (t *TaskDeleteTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return tools.ErrorResult("需要提供 task_id 参数")
	}

	task, err := t.store.GetByID(taskID)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("查询任务失败: %v", err))
	}

	if err := t.store.Delete(taskID); err != nil {
		return tools.ErrorResult(fmt.Sprintf("删除任务失败: %v", err))
	}

	if err := deleteSchedulerTask(t.scheduler, taskID); err != nil {
		t.logger.Warn("删除调度器任务失败", "task_id", taskID, "error", err)
	}

	output := fmt.Sprintf("任务已删除\n\n名称: %s\nID: %s\n调度: %s\n通道: %s",
		task.Name, task.ID, task.CronExpr, task.Channel)

	return tools.SuccessResult(output)
}
