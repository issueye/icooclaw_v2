// Package script provides JavaScript scripting engine for icooclaw.
package script

import (
	"fmt"
	"log/slog"

	appstorage "icooclaw/pkg/storage"
)

// StorageBuiltin exposes a safe, read-only subset of storage functionality.
type StorageBuiltin struct {
	storage *appstorage.Storage
	logger  *slog.Logger
}

// NewStorageBuiltin creates a new storage builtin.
func NewStorageBuiltin(storage *appstorage.Storage, logger *slog.Logger) *StorageBuiltin {
	if logger == nil {
		logger = slog.Default()
	}
	return &StorageBuiltin{
		storage: storage,
		logger:  logger,
	}
}

// Name returns the builtin name.
func (s *StorageBuiltin) Name() string {
	return "storage"
}

// Object returns the storage object.
func (s *StorageBuiltin) Object() map[string]any {
	return map[string]any{
		"param": map[string]any{
			"get":         s.GetParam,
			"list":        s.ListParams,
			"listByGroup": s.ListParamsByGroup,
		},
		"session": map[string]any{
			"get":               s.GetSession,
			"getByChannel":      s.GetSessionByChannel,
			"getByChannelAndID": s.GetSessionByChannel,
		},
		"message": map[string]any{
			"get":              s.GetMessages,
			"getByID":          s.GetMessageByID,
			"getSummary":       s.GetSummaryMessages,
			"getRecentSummary": s.GetRecentSummaryMessages,
			"getLastUser":      s.GetLastUserMessage,
		},
		"task": map[string]any{
			"get":         s.GetTask,
			"list":        s.ListTasks,
			"listEnabled": s.ListEnabledTasks,
		},
	}
}

func (s *StorageBuiltin) requireStorage() error {
	if s.storage == nil {
		return fmt.Errorf(errStorageNotConfigured)
	}
	return nil
}

func (s *StorageBuiltin) GetParam(key string) (*appstorage.ParamConfig, error) {
	if err := s.requireStorage(); err != nil {
		return nil, err
	}
	return s.storage.Param().Get(key)
}

func (s *StorageBuiltin) ListParams() ([]*appstorage.ParamConfig, error) {
	if err := s.requireStorage(); err != nil {
		return nil, err
	}
	return s.storage.Param().List()
}

func (s *StorageBuiltin) ListParamsByGroup(group string) ([]*appstorage.ParamConfig, error) {
	if err := s.requireStorage(); err != nil {
		return nil, err
	}
	return s.storage.Param().ListByGroup(group)
}

func (s *StorageBuiltin) GetSession(id string) (*appstorage.Session, error) {
	if err := s.requireStorage(); err != nil {
		return nil, err
	}
	return s.storage.Session().Get(id)
}

func (s *StorageBuiltin) GetSessionByChannel(channel, sessionID string) (*appstorage.Session, error) {
	if err := s.requireStorage(); err != nil {
		return nil, err
	}
	return s.storage.Session().GetBySessionID(channel, sessionID)
}

func (s *StorageBuiltin) GetMessages(sessionID string, limit int) ([]*appstorage.Message, error) {
	if err := s.requireStorage(); err != nil {
		return nil, err
	}
	return s.storage.Message().Get(sessionID, limit)
}

func (s *StorageBuiltin) GetMessageByID(id string) (*appstorage.Message, error) {
	if err := s.requireStorage(); err != nil {
		return nil, err
	}
	return s.storage.Message().GetByID(id)
}

func (s *StorageBuiltin) GetSummaryMessages(sessionID string) ([]*appstorage.Message, error) {
	if err := s.requireStorage(); err != nil {
		return nil, err
	}
	return s.storage.Message().GetSummary(sessionID)
}

func (s *StorageBuiltin) GetRecentSummaryMessages(sessionID string, limit int) ([]*appstorage.Message, error) {
	if err := s.requireStorage(); err != nil {
		return nil, err
	}
	return s.storage.Message().GetRecentSummary(sessionID, limit)
}

func (s *StorageBuiltin) GetLastUserMessage(sessionID string) (*appstorage.Message, error) {
	if err := s.requireStorage(); err != nil {
		return nil, err
	}
	return s.storage.Message().GetLastUserMessage(sessionID)
}

func (s *StorageBuiltin) GetTask(id string) (*appstorage.Task, error) {
	if err := s.requireStorage(); err != nil {
		return nil, err
	}
	return s.storage.Task().GetByID(id)
}

func (s *StorageBuiltin) ListTasks() ([]appstorage.Task, error) {
	if err := s.requireStorage(); err != nil {
		return nil, err
	}
	return s.storage.Task().GetAll()
}

func (s *StorageBuiltin) ListEnabledTasks() ([]appstorage.Task, error) {
	if err := s.requireStorage(); err != nil {
		return nil, err
	}
	return s.storage.Task().GetEnabled()
}
