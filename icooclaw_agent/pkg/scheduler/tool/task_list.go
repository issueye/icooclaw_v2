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

// TaskListTool 定时任务列表工具
type TaskListTool struct {
	scheduler *scheduler.Scheduler
	store     *storage.TaskStorage
	logger    *slog.Logger
	bus       *bus.MessageBus
}

// NewTaskListTool 创建定时任务列表工具
func NewTaskListTool(store *storage.TaskStorage, scheduler *scheduler.Scheduler, bus *bus.MessageBus, logger *slog.Logger) *TaskListTool {
	if logger == nil {
		logger = slog.Default()
	}
	return &TaskListTool{
		store:     store,
		scheduler: scheduler,
		logger:    logger,
		bus:       bus,
	}
}

// Name 工具名称
func (t *TaskListTool) Name() string {
	return "task_list"
}

// Description 工具描述
func (t *TaskListTool) Description() string {
	return "列出所有已创建的定时任务，支持分页和关键词搜索。\n\n使用场景:\n- 查看当前所有定时任务\n- 搜索特定名称的任务\n- 获取任务总数和状态概览\n\n返回信息包括: 任务名称、调度时间、状态(启用/禁用)、上次运行时间、下次运行时间"
}

// Parameters 工具参数
func (t *TaskListTool) Parameters() map[string]any {
	return map[string]any{
		"page": map[string]any{
			"type":        "integer",
			"description": "页码，从 1 开始 (不提供则返回所有)",
		},
		"page_size": map[string]any{
			"type":        "integer",
			"description": "每页数量，最大 100 (与 page 配合使用)",
		},
		"keyword": map[string]any{
			"type":        "string",
			"description": "搜索关键词，将匹配任务名称和描述",
		},
	}
}

// Execute 执行列出任务
func (t *TaskListTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	page, _ := args["page"].(float64)
	pageSize, _ := args["page_size"].(float64)
	keyword, _ := args["keyword"].(string)

	query := &storage.QueryTask{
		KeyWord: keyword,
	}

	if page > 0 && pageSize > 0 {
		query.Page = storage.Page{
			Page: int(page),
			Size: int(pageSize),
		}
	}

	result, err := t.store.Page(query)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("查询任务失败: %v", err))
	}

	var output string
	if len(result.Records) == 0 {
		output = "没有找到定时任务"
	} else {
		output = fmt.Sprintf("找到 %d 个定时任务 (共 %d 个):\n\n", len(result.Records), result.Page.Total)
		for _, task := range result.Records {
			status := "已启用"
			if !task.Enabled {
				status = "已禁用"
			}
			output += fmt.Sprintf("名称: %s\n", task.Name)
			output += fmt.Sprintf("ID: %s\n", task.ID)
			output += fmt.Sprintf("  类型: %s\n", task.TaskType)
			if task.CronExpr != "" {
				output += fmt.Sprintf("  调度: %s\n", task.CronExpr)
			}
			if task.Description != "" {
				output += fmt.Sprintf("  描述: %s\n", task.Description)
			}
			if task.Content != "" {
				output += fmt.Sprintf("  内容: %s\n", task.Content)
			}
			output += fmt.Sprintf("  状态: %s\n", status)
			if task.LastStatus != "" {
				output += fmt.Sprintf("  最近结果: %s\n", task.LastStatus)
			}
			if task.RunCount > 0 {
				output += fmt.Sprintf("  执行次数: %d\n", task.RunCount)
			}
			if task.LastError != "" {
				output += fmt.Sprintf("  最近错误: %s\n", task.LastError)
			}
			if task.Result != "" {
				output += fmt.Sprintf("  最近结果: %s\n", task.Result)
			}
			if task.LastRunAt != "" {
				output += fmt.Sprintf("  上次运行: %s\n", task.LastRunAt)
			}
			if task.NextRunAt != "" {
				output += fmt.Sprintf("  下次运行: %s\n", task.NextRunAt)
			}
			output += "\n"
		}
	}

	if result.Page.Total > 0 {
		output += fmt.Sprintf("总计: %d 个任务", result.Page.Total)
	}

	return tools.SuccessResult(output)
}
