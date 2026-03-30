package storage

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Task represents a scheduled task.
type Task struct {
	Model
	Name        string `gorm:"column:name;type:varchar(100);not null;comment:任务名称" json:"name"`                   // 任务名称
	Description string `gorm:"column:description;type:text;comment:任务描述" json:"description"`                      // 任务描述
	TaskType    string `gorm:"column:task_type;type:varchar(32);default:scheduled;comment:任务类型" json:"task_type"` // 任务类型
	Executor    string `gorm:"column:executor;type:varchar(64);default:message;comment:执行器" json:"executor"`      // 执行器
	Content     string `gorm:"column:content;type:text;comment:任务内容" json:"content"`                              // 任务内容 (消息文本)
	Channel     string `gorm:"column:channel;type:varchar(100);comment:通道名称" json:"channel"`                      // 发送消息的通道名称
	SessionID   string `gorm:"column:session_id;type:varchar(100);comment:会话ID" json:"session_id"`                // 会话ID
	CronExpr    string `gorm:"column:cron_expr;type:varchar(100);comment:Cron表达式" json:"cron_expr"`               // Cron表达式
	Params      string `gorm:"column:params;type:text;comment:参数(JSON格式)" json:"params"`                          // 任务参数
	Enabled     bool   `gorm:"column:enabled;type:tinyint(1);default:true;comment:是否启用" json:"enabled"`           // 是否任务已启用
	LastRunAt   string `gorm:"column:last_run_at;type:datetime;comment:最后执行时间" json:"last_run_at"`                // 上次运行时间
	NextRunAt   string `gorm:"column:next_run_at;type:datetime;comment:下次执行时间" json:"next_run_at"`                // 下次运行时间
	LastStatus  string `gorm:"column:last_status;type:varchar(32);comment:最后执行状态" json:"last_status"`             // 最后执行状态
	LastError   string `gorm:"column:last_error;type:text;comment:最后一次错误" json:"last_error"`                      // 最后一次错误信息
	Result      string `gorm:"column:result;type:text;comment:最近执行结果" json:"result"`                              // 最近执行结果
	RunCount    int    `gorm:"column:run_count;type:int;default:0;comment:执行次数" json:"run_count"`                 // 累计执行次数
}

// TableName returns the table name for Task.
func (Task) TableName() string {
	return tableNamePrefix + "tasks"
}

type QueryTask struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
	Enabled *bool  `json:"enabled"`
}

type ResQueryTask struct {
	Page    Page   `json:"page"`
	Records []Task `json:"records"`
}

type TaskStorage struct {
	db *gorm.DB
}

type TaskRuntimeState struct {
	LastRunAt  string
	NextRunAt  string
	LastStatus string
	LastError  string
	Result     string
	Enabled    bool
	RunCount   int
}

func NewTaskStorage(db *gorm.DB) *TaskStorage {
	return &TaskStorage{db: db}
}

// Create creates a new task.
func (s *TaskStorage) Create(t *Task) error {
	return s.db.Create(t).Error
}

// Update updates a task.
func (s *TaskStorage) Update(t *Task) error {
	return s.db.Save(t).Error
}

// CreateOrUpdate creates or updates a task.
func (s *TaskStorage) CreateOrUpdate(t *Task) error {
	result := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"description", "task_type", "executor", "content", "cron_expr", "channel", "session_id", "params", "enabled", "last_run_at", "next_run_at", "last_status", "last_error", "result", "run_count"}),
	}).Create(t)
	return result.Error
}

// Delete deletes a task by ID.
func (s *TaskStorage) Delete(id string) error {
	result := s.db.Where("id = ?", id).Delete(&Task{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete task: %w", result.Error)
	}
	return nil
}

// GetByID gets a task by ID.
func (s *TaskStorage) GetByID(id string) (*Task, error) {
	var t Task
	result := s.db.Where("id = ?", id).First(&t)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("task not found")
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get task: %w", result.Error)
	}
	return &t, nil
}

// GetAll gets all tasks.
func (s *TaskStorage) GetAll() ([]Task, error) {
	var tasks []Task
	result := s.db.Order("name").Find(&tasks)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", result.Error)
	}
	return tasks, nil
}

// GetEnabled 获取所有启用的任务.
func (s *TaskStorage) GetEnabled() ([]Task, error) {
	var tasks []Task
	result := s.db.Where("enabled = ?", true).Order("name").Find(&tasks)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get enabled tasks: %w", result.Error)
	}
	return tasks, nil
}

// ToggleEnabled 切换任务状态.
func (s *TaskStorage) ToggleEnabled(id string) (*Task, error) {
	var t Task
	result := s.db.Where("id = ?", id).First(&t)
	if result.Error != nil {
		return nil, fmt.Errorf("获取任务失败: %w", result.Error)
	}
	t.Enabled = !t.Enabled
	err := s.db.Save(&t).Error
	if err != nil {
		return nil, fmt.Errorf("切换任务状态失败: %w", err)
	}
	return &t, nil
}

// UpdateRuntimeState 更新任务运行时状态。
func (s *TaskStorage) UpdateRuntimeState(id string, state TaskRuntimeState) error {
	updates := map[string]any{
		"last_run_at": state.LastRunAt,
		"next_run_at": state.NextRunAt,
		"last_status": state.LastStatus,
		"last_error":  state.LastError,
		"result":      state.Result,
		"enabled":     state.Enabled,
		"run_count":   state.RunCount,
	}

	return s.db.Model(&Task{}).Where("id = ?", id).Updates(updates).Error
}

// Page gets tasks with pagination.
func (s *TaskStorage) Page(query *QueryTask) (*ResQueryTask, error) {
	var res ResQueryTask

	qry := s.db.Model(&Task{})

	if query.KeyWord != "" {
		qry = qry.Where("name LIKE ? OR description LIKE ?", "%"+query.KeyWord+"%", "%"+query.KeyWord+"%")
	}

	if query.Enabled != nil {
		qry = qry.Where("enabled = ?", *query.Enabled)
	}

	qry = qry.Order("name")

	result := qry.Count(&res.Page.Total)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to count tasks: %w", result.Error)
	}

	if query.Page.Page == 0 || query.Page.Size == 0 {
		result = qry.Find(&res.Records)
	} else {
		result = qry.Limit(query.Page.Size).
			Offset((query.Page.Page - 1) * query.Page.Size).
			Find(&res.Records)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", result.Error)
	}

	return &res, nil
}
