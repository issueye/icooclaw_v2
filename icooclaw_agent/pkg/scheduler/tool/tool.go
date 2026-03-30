package tool

import (
	"icooclaw/pkg/bus"
	"icooclaw/pkg/scheduler"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
	"log/slog"
)

// RegisterTaskTools 注册定时任务工具
func RegisterTaskTools(store *storage.TaskStorage, scheduler *scheduler.Scheduler, bus *bus.MessageBus, logger *slog.Logger) []tools.Tool {
	return []tools.Tool{
		NewTaskGetTool(store, scheduler, bus, logger),
		NewTaskCreateTool(store, scheduler, bus, logger),
		NewTaskUpdateTool(store, scheduler, bus, logger),
		NewTaskDeleteTool(store, scheduler, bus, logger),
		NewTaskRunTool(store, scheduler, bus, logger),
		NewTaskEnableTool(store, scheduler, bus, logger),
		NewTaskDisableTool(store, scheduler, bus, logger),
		NewTaskListTool(store, scheduler, bus, logger),
	}
}
