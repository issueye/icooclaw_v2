package tool

import (
	"context"
	"fmt"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/scheduler"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
	"log/slog"
	"time"
)

// TaskGetTool 获取定时任务详情工具
type TaskGetTool struct {
	scheduler *scheduler.Scheduler
	store     *storage.TaskStorage
	logger    *slog.Logger
	bus       *bus.MessageBus
}

// NewTaskGetTool 创建获取定时任务详情工具
func NewTaskGetTool(store *storage.TaskStorage, scheduler *scheduler.Scheduler, bus *bus.MessageBus, logger *slog.Logger) *TaskGetTool {
	if logger == nil {
		logger = slog.Default()
	}
	return &TaskGetTool{
		store:     store,
		scheduler: scheduler,
		logger:    logger,
		bus:       bus,
	}
}

// Name 工具名称
func (t *TaskGetTool) Name() string {
	return "task_get"
}

// Description 工具描述
func (t *TaskGetTool) Description() string {
	return "获取单个定时任务的完整详细信息。\n\n返回信息包括:\n- 任务名称、描述、调度表达式\n- 当前状态 (启用/禁用)\n- 关联的通道和会话 ID\n- 任务参数 (JSON 格式)\n- 创建和更新时间\n- 上次运行时间和下次计划运行时间"
}

// Parameters 工具参数
func (t *TaskGetTool) Parameters() map[string]any {
	return map[string]any{
		"task_id": map[string]any{
			"type":        "string",
			"description": "要查询的任务 ID (通过 task_list 获取)",
		},
		"required": []string{"task_id"},
	}
}

// Execute 执行获取任务详情
func (t *TaskGetTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return tools.ErrorResult("需要提供 task_id 参数")
	}

	task, err := t.store.GetByID(taskID)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("获取任务失败: %v", err))
	}

	status := "已启用"
	if !task.Enabled {
		status = "已禁用"
	}

	output := fmt.Sprintf("任务详情\n\n名称: %s\nID: %s\n类型: %s\n调度: %s\n状态: %s\n通道: %s", task.Name, task.ID, task.TaskType, task.CronExpr, status, task.Channel)
	if task.SessionID != "" {
		output += fmt.Sprintf("\n会话ID: %s", task.SessionID)
	}
	if task.Description != "" {
		output += fmt.Sprintf("\n描述: %s", task.Description)
	}
	if task.Params != "" {
		output += fmt.Sprintf("\n参数: %s", task.Params)
	}
	if task.Content != "" {
		output += fmt.Sprintf("\n内容: %s", task.Content)
	}
	if task.LastRunAt != "" {
		output += fmt.Sprintf("\n上次运行: %s", task.LastRunAt)
	}
	if task.NextRunAt != "" {
		output += fmt.Sprintf("\n下次运行: %s", task.NextRunAt)
	}
	if task.LastStatus != "" {
		output += fmt.Sprintf("\n最近结果: %s", task.LastStatus)
	}
	if task.RunCount > 0 {
		output += fmt.Sprintf("\n执行次数: %d", task.RunCount)
	}
	if task.LastError != "" {
		output += fmt.Sprintf("\n最近错误: %s", task.LastError)
	}
	if task.Result != "" {
		output += fmt.Sprintf("\n最近结果: %s", task.Result)
	}
	output += fmt.Sprintf("\n创建时间: %s", task.CreatedAt.Format(time.RFC3339))
	output += fmt.Sprintf("\n更新时间: %s", task.UpdatedAt.Format(time.RFC3339))

	return tools.SuccessResult(output)
}
