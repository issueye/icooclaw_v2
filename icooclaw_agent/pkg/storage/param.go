package storage

import (
	"fmt"

	"gorm.io/gorm"
)

// ParamConfig 运行时参数配置模型
type ParamConfig struct {
	Model
	Key         string `gorm:"column:key;type:varchar(100);not null;comment:参数键" json:"key"`              // 参数键
	Value       string `gorm:"column:value;type:text;comment:参数值(JSON格式)" json:"value"`                   // 参数值（JSON 格式）
	Description string `gorm:"column:description;type:varchar(500);comment:参数描述" json:"description"`      // 参数描述
	Group       string `gorm:"column:group;type:varchar(50);default:'general';comment:参数分组" json:"group"` // 参数分组
	Enabled     bool   `gorm:"column:enabled;type:tinyint(1);default:true;comment:是否启用" json:"enabled"`   // 是否启用
}

// TableName returns the table name for ParamConfig.
func (ParamConfig) TableName() string {
	return tableNamePrefix + "param_config"
}

type ParamStorage struct {
	db *gorm.DB
}

func NewParamStorage(db *gorm.DB) *ParamStorage {
	return &ParamStorage{db: db}
}

// Save saves a param configuration.
func (s *ParamStorage) Save(p *ParamConfig) error {
	if p == nil {
		return fmt.Errorf("param config is nil")
	}
	return s.db.Create(p).Error
}

// SaveOrUpdateByKey saves a param config, updating the existing record when the key already exists.
func (s *ParamStorage) SaveOrUpdateByKey(p *ParamConfig) error {
	if p == nil {
		return fmt.Errorf("param config is nil")
	}

	existing, err := s.Get(p.Key)
	if err != nil {
		return err
	}
	if existing != nil {
		p.ID = existing.ID
		return s.db.Save(p).Error
	}
	return s.db.Create(p).Error
}

// Get gets a param by key.
func (s *ParamStorage) Get(key string) (*ParamConfig, error) {
	var p ParamConfig
	result := s.db.Where("key = ?", key).First(&p)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get param: %w", result.Error)
	}
	return &p, nil
}

// List lists all param configurations.
func (s *ParamStorage) List() ([]*ParamConfig, error) {
	var params []*ParamConfig
	if err := listOrdered(s.db.Model(&ParamConfig{}), "key", &params, "params"); err != nil {
		return nil, err
	}
	return params, nil
}

// ListByGroup lists all param configurations by group.
func (s *ParamStorage) ListByGroup(group string) ([]*ParamConfig, error) {
	var params []*ParamConfig
	if err := listOrdered(s.db.Model(&ParamConfig{}).Where("group = ?", group), "key", &params, "params by group"); err != nil {
		return nil, err
	}
	return params, nil
}

// Delete deletes a param by key.
func (s *ParamStorage) Delete(key string) error {
	return deleteByField(s.db, "key", key, &ParamConfig{}, "param")
}

type QueryParam struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
	Group   string `json:"group"`
	Enabled *bool  `json:"enabled"`
}

type ResQueryParam struct {
	Page    Page          `json:"page"`
	Records []ParamConfig `json:"records"`
}

// Page gets param configurations with pagination.
func (s *ParamStorage) Page(query *QueryParam) (*ResQueryParam, error) {
	var res ResQueryParam

	qry := s.db.Model(&ParamConfig{})

	if query.KeyWord != "" {
		qry = qry.Where("key LIKE ? OR description LIKE ?", "%"+query.KeyWord+"%", "%"+query.KeyWord+"%")
	}

	if query.Group != "" {
		qry = qry.Where("group = ?", query.Group)
	}

	if query.Enabled != nil {
		qry = qry.Where("enabled = ?", *query.Enabled)
	}

	page, err := pageQuery(qry, "key", query.Page, &res.Records, "params")
	if err != nil {
		return nil, err
	}
	res.Page = page

	return &res, nil
}
