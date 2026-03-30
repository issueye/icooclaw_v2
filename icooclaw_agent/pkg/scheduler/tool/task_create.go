package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/scheduler"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
	"log/slog"
	"strings"
)

// TaskCreateTool 定时任务创建工具
type TaskCreateTool struct {
	scheduler *scheduler.Scheduler
	store     *storage.TaskStorage
	logger    *slog.Logger
	bus       *bus.MessageBus
}

// NewTaskCreateTool 创建定时任务工具
func NewTaskCreateTool(store *storage.TaskStorage, scheduler *scheduler.Scheduler, bus *bus.MessageBus, logger *slog.Logger) *TaskCreateTool {
	if logger == nil {
		logger = slog.Default()
	}
	return &TaskCreateTool{
		store:     store,
		scheduler: scheduler,
		logger:    logger,
		bus:       bus,
	}
}

// Name 工具名称
func (t *TaskCreateTool) Name() string {
	return "task_create"
}

// Description 工具描述
func (t *TaskCreateTool) Description() string {
	return "创建一个任务，支持立即执行和定时执行两种类型。\n\n使用场景:\n- 立即向指定会话投递一次任务\n- 按 cron 表达式定时发送消息\n- 自动化的日常任务调度\n\nCron 表达式示例:\n- '0 * * * *' - 每小时整点\n- '*/5 * * * *' - 每 5 分钟\n- '0 9 * * *' - 每天上午 9 点\n- '0 9 * * 1-5' - 工作日上午 9 点"
}

// Parameters 工具参数
func (t *TaskCreateTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "任务的唯一标识名称，便于识别和管理",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "任务的详细描述，说明任务的用途和功能",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "任务执行时发送的消息内容 (文本)",
			},
			"task_type": map[string]any{
				"type":        "string",
				"description": "任务类型，immediate=立即执行，scheduled=定时执行，默认 scheduled",
			},
			"cron_expr": map[string]any{
				"type":        "string",
				"description": "定时执行任务使用的 Cron 时间表达式，格式: 分 时 日 月 周",
			},
			"channel": map[string]any{
				"type":        "string",
				"description": "消息通道类型，如 'websocket', 'qq', 'feishu', 'dingtalk'",
			},
			"session_id": map[string]any{
				"type":        "string",
				"description": "会话 ID (用户 ID 或群组 ID)，消息将发送到此处",
			},
			"params": map[string]any{
				"type":        "string",
				"description": "任务执行的额外参数，JSON 格式 (如: '{\"key\":\"value\"}')，会与 content 合并发送",
			},
			"enabled": map[string]any{
				"type":        "boolean",
				"description": "创建后是否立即启用，true=启用调度 false=仅保存不调度",
			},
		},
		"required": []string{"name", "description", "content", "channel", "session_id"},
	}
}

func (t *TaskCreateTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	name, _ := args["name"].(string)
	if name == "" {
		return tools.ErrorResult("需要提供 name 参数")
	}

	description, _ := args["description"].(string)
	if description == "" {
		return tools.ErrorResult("需要提供 description 参数")
	}

	content, _ := args["content"].(string)
	if content == "" {
		return tools.ErrorResult("需要提供 content 参数")
	}

	taskType := scheduler.TaskTypeScheduled
	if providedTaskType, _ := args["task_type"].(string); strings.TrimSpace(providedTaskType) != "" {
		taskType = strings.TrimSpace(providedTaskType)
	}
	if taskType != scheduler.TaskTypeImmediate && taskType != scheduler.TaskTypeScheduled {
		return tools.ErrorResult("task_type 只能是 immediate 或 scheduled")
	}

	cronExpr, _ := args["cron_expr"].(string)
	channel, _ := args["channel"].(string)
	if channel == "" {
		return tools.ErrorResult("需要提供 channel 参数")
	}

	sessionID, _ := args["session_id"].(string)
	if sessionID == "" {
		return tools.ErrorResult("需要提供 session_id 参数")
	}

	if taskType == scheduler.TaskTypeScheduled {
		if cronExpr == "" {
			return tools.ErrorResult("scheduled 类型任务需要提供 cron_expr")
		}
		if err := validateCronExpr(cronExpr); err != nil {
			return tools.ErrorResult(fmt.Sprintf("无效的 Cron 表达式: %v", err))
		}
	}
	normalizedCronExpr, err := normalizeCronExpr(cronExpr)
	if err != nil && taskType == scheduler.TaskTypeScheduled {
		return tools.ErrorResult(fmt.Sprintf("无效的 Cron 表达式: %v", err))
	}
	if taskType != scheduler.TaskTypeScheduled {
		normalizedCronExpr = ""
	}

	params, _ := args["params"].(string)
	enabled := true
	if e, ok := args["enabled"]; ok {
		switch v := e.(type) {
		case bool:
			enabled = v
		case string:
			enabled = v == "true"
		}
	}

	if params != "" {
		if !json.Valid([]byte(params)) {
			return tools.ErrorResult("params 必须是有效的 JSON 格式")
		}
	}

	task := &storage.Task{
		Name:        name,
		Description: description,
		TaskType:    taskType,
		Executor:    scheduler.TaskExecutorMessage,
		Content:     content,
		Channel:     channel,
		SessionID:   sessionID,
		CronExpr:    normalizedCronExpr,
		Params:      params,
		Enabled:     enabled,
	}

	if err := t.store.Create(task); err != nil {
		return tools.ErrorResult(fmt.Sprintf("创建任务失败: %v", err))
	}

	if err := syncSchedulerTask(t.scheduler, task); err != nil {
		return tools.ErrorResult(fmt.Sprintf("同步调度器失败: %v", err))
	}

	enabledStr := "已启用"
	if !task.Enabled {
		enabledStr = "已禁用"
	}
	output := fmt.Sprintf("任务创建成功\n\n名称: %s\nID: %s\n类型: %s\n内容: %s\n调度: %s\n通道: %s\n状态: %s",
		task.Name, task.ID, task.TaskType, task.Content, task.CronExpr, task.Channel, enabledStr)

	return tools.SuccessResult(output)
}
