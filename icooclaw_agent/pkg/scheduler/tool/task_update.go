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
)

// TaskUpdateTool 定时任务更新工具
type TaskUpdateTool struct {
	scheduler *scheduler.Scheduler
	store     *storage.TaskStorage
	logger    *slog.Logger
	bus       *bus.MessageBus
}

// NewTaskUpdateTool 创建定时任务更新工具
func NewTaskUpdateTool(store *storage.TaskStorage, scheduler *scheduler.Scheduler, bus *bus.MessageBus, logger *slog.Logger) *TaskUpdateTool {
	if logger == nil {
		logger = slog.Default()
	}
	return &TaskUpdateTool{
		store:     store,
		scheduler: scheduler,
		logger:    logger,
		bus:       bus,
	}
}

// Name 工具名称
func (t *TaskUpdateTool) Name() string {
	return "task_update"
}

// Description 工具描述
func (t *TaskUpdateTool) Description() string {
	return "更新已存在的任务配置。\n\n可更新的字段:\n- name: 任务名称\n- description: 任务描述\n- task_type: immediate 或 scheduled\n- content: 任务内容 (消息文本)\n- cron_expr: 定时任务的调度时间表达式\n- channel: 消息通道类型\n- session_id: 会话 ID\n- params: 任务参数 (JSON)\n- enabled: 是否启用\n\n注意: 更新为定时执行后，下一次执行时间会重新计算"
}

// Parameters 工具参数
func (t *TaskUpdateTool) Parameters() map[string]any {
	return map[string]any{
		"task_id": map[string]any{
			"type":        "string",
			"description": "要更新的任务 ID (必需)",
		},
		"name": map[string]any{
			"type":        "string",
			"description": "新的任务名称",
		},
		"description": map[string]any{
			"type":        "string",
			"description": "新的任务描述",
		},
		"task_type": map[string]any{
			"type":        "string",
			"description": "新的任务类型: immediate 或 scheduled",
		},
		"content": map[string]any{
			"type":        "string",
			"description": "新的任务内容 (消息文本)",
		},
		"cron_expr": map[string]any{
			"type":        "string",
			"description": "新的 Cron 表达式 (例: '0 9 * * *' 每天 9 点)",
		},
		"channel": map[string]any{
			"type":        "string",
			"description": "新的消息通道类型，如 'websocket', 'qq', 'feishu', 'dingtalk'",
		},
		"session_id": map[string]any{
			"type":        "string",
			"description": "新的会话 ID",
		},
		"params": map[string]any{
			"type":        "string",
			"description": "新的任务参数，JSON 格式",
		},
		"enabled": map[string]any{
			"type":        "boolean",
			"description": "是否启用任务",
		},
		"required": []string{"task_id"},
	}
}

func (t *TaskUpdateTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return tools.ErrorResult("需要提供 task_id 参数")
	}

	name, _ := args["name"].(string)
	description, _ := args["description"].(string)
	taskType, _ := args["task_type"].(string)
	content, _ := args["content"].(string)
	cronExpr, _ := args["cron_expr"].(string)
	channel, _ := args["channel"].(string)
	sessionID, _ := args["session_id"].(string)
	params, _ := args["params"].(string)

	if cronExpr != "" {
		if err := validateCronExpr(cronExpr); err != nil {
			return tools.ErrorResult(fmt.Sprintf("无效的 Cron 表达式: %v", err))
		}
		normalizedCronExpr, err := normalizeCronExpr(cronExpr)
		if err != nil {
			return tools.ErrorResult(fmt.Sprintf("无效的 Cron 表达式: %v", err))
		}
		cronExpr = normalizedCronExpr
	}

	if params != "" {
		if !json.Valid([]byte(params)) {
			return tools.ErrorResult("params 必须是有效的 JSON 格式")
		}
	}

	task, err := t.store.GetByID(taskID)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("获取任务失败: %v", err))
	}
	if taskType != "" && taskType != scheduler.TaskTypeImmediate && taskType != scheduler.TaskTypeScheduled {
		return tools.ErrorResult("task_type 只能是 immediate 或 scheduled")
	}

	if name != "" {
		task.Name = name
	}
	if description != "" {
		task.Description = description
	}
	if taskType != "" {
		task.TaskType = taskType
	}
	if content != "" {
		task.Content = content
	}
	if cronExpr != "" {
		task.CronExpr = cronExpr
	}
	if channel != "" {
		task.Channel = channel
	}
	if sessionID != "" {
		task.SessionID = sessionID
	}
	if params != "" {
		task.Params = params
	}
	if e, ok := args["enabled"]; ok {
		switch v := e.(type) {
		case bool:
			task.Enabled = v
		case string:
			task.Enabled = v == "true"
		}
	}
	if task.TaskType == scheduler.TaskTypeScheduled && task.CronExpr == "" {
		return tools.ErrorResult("scheduled 类型任务需要提供 cron_expr")
	}
	if task.TaskType == scheduler.TaskTypeImmediate {
		task.CronExpr = ""
	}

	if err := t.store.Update(task); err != nil {
		return tools.ErrorResult(fmt.Sprintf("更新任务失败: %v", err))
	}

	if err := syncSchedulerTask(t.scheduler, task); err != nil {
		return tools.ErrorResult(fmt.Sprintf("同步调度器失败: %v", err))
	}

	enabledStr := "已启用"
	if !task.Enabled {
		enabledStr = "已禁用"
	}
	output := fmt.Sprintf("任务更新成功\n\n名称: %s\nID: %s\n类型: %s\n内容: %s\n调度: %s\n通道: %s\n状态: %s",
		task.Name, task.ID, task.TaskType, task.Content, task.CronExpr, task.Channel, enabledStr)

	return tools.SuccessResult(output)
}
