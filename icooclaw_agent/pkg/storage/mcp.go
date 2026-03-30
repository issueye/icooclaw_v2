package storage

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MCPType string

const (
	MCPTypeStdio MCPType = "stdio" // stdio 类型的 MCP
	MCPTypeSSE   MCPType = "sse"   // sse 类型的 MCP
)

func (mcpType MCPType) String() string {
	return string(mcpType)
}

type MCPConfig struct {
	Model                            // 嵌入 Model 结构体
	Name           string            `gorm:"column:name;type:varchar(100);uniqueIndex;not null;comment:MCP名称" json:"name"`      // MCP 名称
	Description    string            `gorm:"column:description;type:varchar(255);comment:MCP描述" json:"description"`             // MCP 描述
	Type           MCPType           `gorm:"column:type;type:varchar(100);not null;comment:MCP类型(stdio/sse)" json:"type"`       // MCP 类型
	Enabled        bool              `gorm:"column:enabled;type:tinyint(1);default:true;comment:是否启用" json:"enabled"`           // 是否启用
	Command        string            `gorm:"column:command;type:text;comment:stdio命令" json:"command"`                           // stdio 命令
	URL            string            `gorm:"column:url;type:text;comment:SSE地址" json:"url"`                                     // SSE 地址
	Args           StringArray       `gorm:"column:args;type:text;serializer:json;comment:MCP参数(JSON数组)" json:"args"`           // MCP 参数
	Env            map[string]string `gorm:"column:env;type:text;serializer:json;comment:环境变量(JSON对象)" json:"env"`              // 环境变量
	Headers        map[string]string `gorm:"column:headers;type:text;serializer:json;comment:SSE请求头(JSON对象)" json:"headers"`    // SSE 请求头
	RetryCount     int               `gorm:"column:retry_count;type:int;default:3;comment:重试次数" json:"retry_count"`             // 重试次数
	TimeoutSeconds int               `gorm:"column:timeout_seconds;type:int;default:30;comment:超时时间(秒)" json:"timeout_seconds"` // 超时时间（秒）
}

func (table *MCPConfig) IsStdio() bool {
	return table.Type == MCPTypeStdio
}

func (table *MCPConfig) IsSSE() bool {
	return table.Type == MCPTypeSSE || strings.EqualFold(table.Type.String(), "Streamable HTTP")
}

func (table *MCPConfig) ArgsString() string {
	return strings.Join(table.Args, " ")
}

func (table *MCPConfig) TableName() string {
	return tableNamePrefix + "mcp"
}

type MCPStorage struct {
	db *gorm.DB
}

func NewMCPStorage(db *gorm.DB) *MCPStorage {
	return &MCPStorage{db: db}
}

// BeforeCreate 创建前回调
func (c *MCPConfig) BeforeCreate(tx *gorm.DB) error {
	c.ID = uuid.New().String()
	return nil
}

// Save saves a MCP configuration.
func (s *MCPStorage) Save(m *MCPConfig) error {
	if strings.TrimSpace(m.ID) == "" {
		return s.Create(m)
	}
	return s.Update(m)
}

// Get gets a MCP by name.
func (s *MCPStorage) Get(name string) (*MCPConfig, error) {
	var m MCPConfig
	result := s.db.Where("name = ?", name).First(&m)
	if result.Error != nil {
		return nil, result.Error
	}
	return &m, nil
}

// List lists all MCP configurations.
func (s *MCPStorage) List() ([]*MCPConfig, error) {
	var mcpConfigs []*MCPConfig
	if err := listOrdered(s.db.Model(&MCPConfig{}), "name", &mcpConfigs, "mcp configs"); err != nil {
		return nil, err
	}
	return mcpConfigs, nil
}

// ListEnabled 列出所有启用的 MCP 配置。
func (s *MCPStorage) ListEnabled() ([]*MCPConfig, error) {
	var mcpConfigs []*MCPConfig
	if err := listOrdered(s.db.Model(&MCPConfig{}).Where("enabled = ?", true), "name", &mcpConfigs, "enabled mcp configs"); err != nil {
		return nil, err
	}
	return mcpConfigs, nil
}

// Delete deletes a MCP by name.
func (s *MCPStorage) Delete(name string) error {
	return deleteByField(s.db, "name", name, &MCPConfig{}, "mcp config")
}

// DeleteByID deletes a MCP by ID.
func (s *MCPStorage) DeleteByID(id string) error {
	return deleteByField(s.db, "id", id, &MCPConfig{}, "mcp config")
}

// GetByID gets a MCP by ID.
func (s *MCPStorage) GetByID(id string) (*MCPConfig, error) {
	var m MCPConfig
	result := s.db.Where("id = ?", id).First(&m)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("mcp config not found")
		}
		return nil, fmt.Errorf("failed to get mcp config: %w", result.Error)
	}
	return &m, nil
}

// Update updates a MCP configuration.
func (s *MCPStorage) Update(m *MCPConfig) error {
	result := s.db.Save(m)
	if result.Error != nil {
		return fmt.Errorf("failed to update mcp config: %w", result.Error)
	}
	return nil
}

// Create creates a new MCP configuration.
func (s *MCPStorage) Create(m *MCPConfig) error {
	return s.db.Create(m).Error
}

type QueryMCP struct {
	Page    Page    `json:"page"`
	KeyWord string  `json:"key_word"`
	Type    MCPType `json:"type"`
}

type ResQueryMCP struct {
	Page    Page        `json:"page"`
	Records []MCPConfig `json:"records"`
}

// Page gets MCP configurations with pagination.
func (s *MCPStorage) Page(query *QueryMCP) (*ResQueryMCP, error) {
	var res ResQueryMCP

	qry := s.db.Model(&MCPConfig{})

	if query.KeyWord != "" {
		qry = qry.Where("name LIKE ? OR description LIKE ?", "%"+query.KeyWord+"%", "%"+query.KeyWord+"%")
	}

	if query.Type != "" {
		qry = qry.Where("type = ?", query.Type)
	}

	page, err := pageQuery(qry, "name", query.Page, &res.Records, "mcp configs")
	if err != nil {
		return nil, err
	}
	res.Page = page

	return &res, nil
}
