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

// TaskRunTool 定时任务立即执行工具
type TaskRunTool struct {
	scheduler *scheduler.Scheduler
	store     *storage.TaskStorage
	logger    *slog.Logger
	bus       *bus.MessageBus
}

// NewTaskRunTool 创建立即执行定时任务工具
func NewTaskRunTool(store *storage.TaskStorage, scheduler *scheduler.Scheduler, bus *bus.MessageBus, logger *slog.Logger) *TaskRunTool {
	if logger == nil {
		logger = slog.Default()
	}
	return &TaskRunTool{
		store:     store,
		scheduler: scheduler,
		logger:    logger,
		bus:       bus,
	}
}

// Name 工具名称
func (t *TaskRunTool) Name() string {
	return "task_run"
}

// Description 工具描述
func (t *TaskRunTool) Description() string {
	return "立即触发执行一个定时任务，与正常的调度时间无关。\n\n特点:\n- 立即执行，不等待下次调度时间\n- 执行后不影响原有的调度计划\n- 可用于测试任务配置是否正确\n- 仅支持执行已启用的任务\n\n使用场景:\n- 测试新创建的任务\n- 手动触发一次任务执行"
}

// Parameters 工具参数
func (t *TaskRunTool) Parameters() map[string]any {
	return map[string]any{
		"task_id": map[string]any{
			"type":        "string",
			"description": "要立即执行的任务 ID",
		},
		"required": []string{"task_id"},
	}
}

// Execute 执行立即运行任务
func (t *TaskRunTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return tools.ErrorResult("需要提供 task_id 参数")
	}

	if t.scheduler == nil {
		return tools.ErrorResult("调度器未初始化")
	}

	if _, err := t.scheduler.RunTask(taskID); err != nil {
		return tools.ErrorResult(fmt.Sprintf("执行任务失败: %v", err))
	}

	return tools.SuccessResult(fmt.Sprintf("任务已触发执行\n\n任务ID: %s\n\n注意: 本次执行不影响原有的调度计划", taskID))
}
