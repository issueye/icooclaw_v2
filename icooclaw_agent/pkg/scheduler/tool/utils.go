package tool

import (
	"icooclaw/pkg/scheduler"
	"icooclaw/pkg/storage"
)

// validateCronExpr 验证 Cron 表达式是否有效.
func validateCronExpr(expr string) error {
	return scheduler.ValidateSchedule(expr)
}

func normalizeCronExpr(expr string) (string, error) {
	return scheduler.NormalizeSchedule(expr)
}

func syncSchedulerTask(s *scheduler.Scheduler, task *storage.Task) error {
	if s == nil || task == nil {
		return nil
	}
	_, err := s.ApplyStorageTask(task)
	return err
}

func deleteSchedulerTask(s *scheduler.Scheduler, taskID string) error {
	if s == nil {
		return nil
	}
	return s.DeleteTask(taskID)
}
