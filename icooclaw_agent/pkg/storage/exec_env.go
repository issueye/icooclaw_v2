package storage

import (
	"fmt"

	"gorm.io/gorm"
)

// ExecEnv stores runtime environment variables separately from generic params.
type ExecEnv struct {
	Model
	Key         string `gorm:"column:key;type:varchar(100);not null;uniqueIndex;comment:环境变量键" json:"key"`
	Value       string `gorm:"column:value;type:text;comment:环境变量值" json:"value"`
	Description string `gorm:"column:description;type:varchar(500);comment:环境变量描述" json:"description"`
	Enabled     bool   `gorm:"column:enabled;type:tinyint(1);default:true;comment:是否启用" json:"enabled"`
}

func (ExecEnv) TableName() string {
	return tableNamePrefix + "exec_env"
}

type ExecEnvStorage struct {
	db *gorm.DB
}

func NewExecEnvStorage(db *gorm.DB) *ExecEnvStorage {
	return &ExecEnvStorage{db: db}
}

func (s *ExecEnvStorage) Save(env *ExecEnv) error {
	if env == nil {
		return fmt.Errorf("exec env is nil")
	}
	return s.db.Create(env).Error
}

func (s *ExecEnvStorage) SaveOrUpdateByKey(env *ExecEnv) error {
	if env == nil {
		return fmt.Errorf("exec env is nil")
	}

	existing, err := s.Get(env.Key)
	if err != nil {
		return err
	}
	if existing != nil {
		env.ID = existing.ID
		return s.db.Save(env).Error
	}
	return s.db.Create(env).Error
}

func (s *ExecEnvStorage) Get(key string) (*ExecEnv, error) {
	var env ExecEnv
	result := s.db.Where("key = ?", key).First(&env)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get exec env: %w", result.Error)
	}
	return &env, nil
}

func (s *ExecEnvStorage) List() ([]*ExecEnv, error) {
	var values []*ExecEnv
	result := s.db.Order("key").Find(&values)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list exec envs: %w", result.Error)
	}
	return values, nil
}

func (s *ExecEnvStorage) Delete(key string) error {
	result := s.db.Where("key = ?", key).Delete(&ExecEnv{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete exec env: %w", result.Error)
	}
	return nil
}

func (s *ExecEnvStorage) ReplaceAll(values map[string]string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&ExecEnv{}).Error; err != nil {
			return fmt.Errorf("failed to clear exec envs: %w", err)
		}
		for key, value := range values {
			if key == "" {
				continue
			}
			if err := tx.Create(&ExecEnv{
				Key:     key,
				Value:   value,
				Enabled: true,
			}).Error; err != nil {
				return fmt.Errorf("failed to save exec env %s: %w", key, err)
			}
		}
		return nil
	})
}

func (s *ExecEnvStorage) ToMap() (map[string]string, error) {
	values, err := s.List()
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(values))
	for _, item := range values {
		if item == nil || !item.Enabled || item.Key == "" {
			continue
		}
		result[item.Key] = item.Value
	}
	return result, nil
}
