package storage

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	icooclawErrors "icooclaw/pkg/errors"
)

// Tool represents a tool configuration.
type Tool struct {
	Model
	Name       string `gorm:"column:name;type:varchar(100);uniqueIndex;not null;comment:工具名称" json:"name"`
	Type       string `gorm:"column:type;type:varchar(50);not null;comment:工具类型(builtin/mcp/custom)" json:"type"` // builtin, mcp, custom
	Definition string `gorm:"column:definition;type:text;comment:工具定义(JSON格式)" json:"definition"`                 // JSON tool definition
	Config     string `gorm:"column:config;type:text;comment:配置(JSON格式)" json:"config"`                           // JSON config
	Enabled    bool   `gorm:"column:enabled;type:tinyint(1);default:true;comment:是否启用" json:"enabled"`
}

// TableName returns the table name for Tool.
func (Tool) TableName() string {
	return tableNamePrefix + "tools"
}

type ToolStorage struct {
	db *gorm.DB
}

func NewToolStorage(db *gorm.DB) *ToolStorage {
	return &ToolStorage{db: db}
}

// SaveTool saves a tool configuration.
func (s *ToolStorage) SaveTool(t *Tool) error {
	result := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"type", "definition", "config", "enabled"}),
	}).Create(t)
	return result.Error
}

// GetTool gets a tool by name.
func (s *ToolStorage) GetTool(name string) (*Tool, error) {
	var t Tool
	result := s.db.Where("name = ?", name).First(&t)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get tool: %w", result.Error)
	}
	return &t, nil
}

// ListTools lists all tools.
func (s *ToolStorage) ListTools() ([]*Tool, error) {
	var tools []*Tool
	if err := listOrdered(s.db.Model(&Tool{}), "name", &tools, "tools"); err != nil {
		return nil, err
	}
	return tools, nil
}

// ListEnabledTools lists all enabled tools.
func (s *ToolStorage) ListEnabledTools() ([]*Tool, error) {
	var tools []*Tool
	if err := listOrdered(s.db.Model(&Tool{}).Where("enabled = ?", true), "name", &tools, "enabled tools"); err != nil {
		return nil, err
	}
	return tools, nil
}

// DeleteTool deletes a tool by name.
func (s *ToolStorage) DeleteTool(name string) error {
	return deleteByField(s.db, "name", name, &Tool{}, "tool")
}

type QueryTool struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
	Type    string `json:"type"`
	Enabled *bool  `json:"enabled"`
}

type ResQueryTool struct {
	Page    Page   `json:"page"`
	Records []Tool `json:"records"`
}

// Page gets tools with pagination.
func (s *ToolStorage) Page(query *QueryTool) (*ResQueryTool, error) {
	var res ResQueryTool

	qry := s.db.Model(&Tool{})

	if query.KeyWord != "" {
		qry = qry.Where("name LIKE ? OR type LIKE ?", "%"+query.KeyWord+"%", "%"+query.KeyWord+"%")
	}

	if query.Type != "" {
		qry = qry.Where("type = ?", query.Type)
	}

	if query.Enabled != nil {
		qry = qry.Where("enabled = ?", *query.Enabled)
	}

	page, err := pageQuery(qry, "name", query.Page, &res.Records, "tools")
	if err != nil {
		return nil, err
	}
	res.Page = page

	return &res, nil
}
