package storage

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"icooclaw/pkg/consts"
	icooclawErrors "icooclaw/pkg/errors"
)

const (
	AgentTypeMaster   = "master"
	AgentTypeSubAgent = "subagent"
)

// Agent represents a persistent agent profile.
type Agent struct {
	Model
	Name         string         `gorm:"column:name;type:varchar(100);uniqueIndex;not null;comment:智能体名称" json:"name"`
	Type         string         `gorm:"column:type;type:varchar(32);default:master;comment:智能体类型" json:"type"`
	Description  string         `gorm:"column:description;type:text;comment:智能体描述" json:"description"`
	SystemPrompt string         `gorm:"column:system_prompt;type:text;comment:额外系统提示词" json:"system_prompt"`
	Enabled      bool           `gorm:"column:enabled;type:tinyint(1);default:true;comment:是否启用" json:"enabled"`
	Metadata     map[string]any `gorm:"column:metadata;type:text;serializer:json;comment:元数据(JSON格式)" json:"metadata"`
}

// TableName returns the table name for Agent.
func (Agent) TableName() string {
	return tableNamePrefix + "agents"
}

type AgentStorage struct {
	db *gorm.DB
}

func NewAgentStorage(db *gorm.DB) *AgentStorage {
	return &AgentStorage{db: db}
}

func NormalizeAgentType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case AgentTypeSubAgent:
		return AgentTypeSubAgent
	default:
		return AgentTypeMaster
	}
}

func IsValidAgentType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", AgentTypeMaster, AgentTypeSubAgent:
		return true
	default:
		return false
	}
}

func (s *AgentStorage) Save(agent *Agent) error {
	if agent == nil {
		return fmt.Errorf("agent is nil")
	}
	agent.Type = NormalizeAgentType(agent.Type)
	if agent.ID != "" {
		return s.db.Save(agent).Error
	}

	result := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"type", "description", "system_prompt", "enabled", "metadata"}),
	}).Create(agent)
	return result.Error
}

func (s *AgentStorage) GetByID(id string) (*Agent, error) {
	var agent Agent
	result := s.db.Where("id = ?", id).First(&agent)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get agent by id: %w", result.Error)
	}
	return &agent, nil
}

func (s *AgentStorage) GetByName(name string) (*Agent, error) {
	var agent Agent
	result := s.db.Where("name = ?", name).First(&agent)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get agent by name: %w", result.Error)
	}
	return &agent, nil
}

func (s *AgentStorage) GetDefault() (*Agent, error) {
	agent, err := s.GetByName(consts.DEFAULT_AGENT_NAME)
	if err == nil {
		return agent, nil
	}
	if err != icooclawErrors.ErrRecordNotFound {
		return nil, err
	}

	defaultAgent := &Agent{
		Name:         consts.DEFAULT_AGENT_NAME,
		Type:         AgentTypeMaster,
		Description:  "默认智能体",
		SystemPrompt: "",
		Enabled:      true,
	}
	if err := s.Save(defaultAgent); err != nil {
		return nil, fmt.Errorf("failed to create default agent: %w", err)
	}
	return s.GetByName(consts.DEFAULT_AGENT_NAME)
}

func (s *AgentStorage) List() ([]*Agent, error) {
	var agents []*Agent
	if err := listOrdered(s.db.Model(&Agent{}), "name", &agents, "agents"); err != nil {
		return nil, err
	}
	return agents, nil
}

func (s *AgentStorage) ListEnabled() ([]*Agent, error) {
	var agents []*Agent
	if err := listOrdered(s.db.Model(&Agent{}).Where("enabled = ?", true), "name", &agents, "enabled agents"); err != nil {
		return nil, err
	}
	return agents, nil
}

func (s *AgentStorage) Delete(id string) error {
	return deleteByField(s.db, "id", id, &Agent{}, "agent")
}

type QueryAgent struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
	Enabled *bool  `json:"enabled"`
	Type    string `json:"type"`
}

type ResQueryAgent struct {
	Page    Page    `json:"page"`
	Records []Agent `json:"records"`
}

func (s *AgentStorage) Page(query *QueryAgent) (*ResQueryAgent, error) {
	var res ResQueryAgent

	qry := s.db.Model(&Agent{})
	if query.KeyWord != "" {
		qry = qry.Where("name LIKE ? OR description LIKE ?", "%"+query.KeyWord+"%", "%"+query.KeyWord+"%")
	}
	if query.Enabled != nil {
		qry = qry.Where("enabled = ?", *query.Enabled)
	}
	if normalizedType := NormalizeAgentType(query.Type); strings.TrimSpace(query.Type) != "" {
		qry = qry.Where("type = ?", normalizedType)
	}

	page, err := pageQuery(qry, "name", query.Page, &res.Records, "agents")
	if err != nil {
		return nil, err
	}
	res.Page = page
	return &res, nil
}
