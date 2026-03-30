package storage

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"icooclaw/pkg/consts"
	icooclawErrors "icooclaw/pkg/errors"
)

type LLMs []LLM

type LLM struct {
	Alias string `gorm:"column:alias;type:varchar(100);not null;comment:模型别名" json:"alias"`
	Model string `gorm:"column:model;type:varchar(100);not null;comment:模型名称" json:"model"`
}

func (s LLM) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *LLMs) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	var str string
	switch v := value.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return nil
	}

	if str == "" {
		*s = nil
		return nil
	}

	return json.Unmarshal([]byte(str), s)
}

// Provider represents a provider configuration.
type Provider struct {
	Model
	Name         string                  `gorm:"column:name;type:varchar(100);not null;comment:提供商名称" json:"name"`              // 提供商名称
	Type         consts.ProviderType     `gorm:"column:type;type:varchar(50);not null;comment:提供商类型" json:"type"`               // 提供商类型
	Protocol     consts.ProviderProtocol `gorm:"column:protocol;type:varchar(50);not null;comment:提供商协议" json:"protocol"`       // 提供商协议
	APIKey       string                  `gorm:"column:api_key;type:varchar(255);comment:API密钥" json:"api_key"`                 // API密钥
	APIBase      string                  `gorm:"column:api_base;type:varchar(255);comment:API基础URL" json:"api_base"`            // API基础URL
	DefaultModel string                  `gorm:"column:default_model;type:varchar(100);comment:默认模型" json:"default_model"`      // 默认模型别名
	LLMs         LLMs                    `gorm:"column:llms;type:text;serializer:json;comment:LLM列表" json:"llms"`               // LLM列表
	Config       string                  `gorm:"column:config;type:text;comment:配置(JSON格式)" json:"config"`                      // JSON object
	Enabled      bool                    `gorm:"column:isenabled;type:boolean;default:false;comment:是否启用" json:"enabled"`       // 是否启用
	Metadata     map[string]any          `gorm:"column:metadata;type:text;serializer:json;comment:元数据(JSON格式)" json:"metadata"` // JSON object
}

// TableName returns the table name for Provider.
func (Provider) TableName() string {
	return tableNamePrefix + "providers"
}

type ProviderStorage struct {
	db *gorm.DB
}

func NewProviderStorage(db *gorm.DB) *ProviderStorage {
	return &ProviderStorage{db: db}
}

// Save saves a provider configuration.
func (s *ProviderStorage) Save(p *Provider) error {
	if err := normalizeProviderProtocol(p); err != nil {
		return err
	}
	result := s.db.Save(p)
	return result.Error
}

// GetByName gets a provider by name.
func (s *ProviderStorage) GetByName(name string) (*Provider, error) {
	var p Provider
	result := s.db.Where("name = ?", name).First(&p)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get provider: %w", result.Error)
	}
	if err := s.persistNormalizedProtocol(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

// List lists all providers.
func (s *ProviderStorage) List() ([]*Provider, error) {
	var providers []*Provider
	if err := listOrdered(s.db.Model(&Provider{}), "name", &providers, "providers"); err != nil {
		return nil, err
	}
	return providers, nil
}

// NormalizeProtocols backfills missing provider protocols using the provider type and legacy minimax metadata.
func (s *ProviderStorage) NormalizeProtocols() error {
	var providers []*Provider
	if err := s.db.Where("protocol IS NULL OR TRIM(protocol) = ''").Find(&providers).Error; err != nil {
		return fmt.Errorf("failed to load providers for protocol normalization: %w", err)
	}

	for _, provider := range providers {
		if err := s.persistNormalizedProtocol(provider); err != nil {
			return err
		}
	}

	return nil
}

// Delete deletes a provider by ID.
func (s *ProviderStorage) Delete(id string) error {
	return deleteByField(s.db, "id", id, &Provider{}, "provider")
}

// DeleteByName deletes a provider by name.
func (s *ProviderStorage) DeleteByName(name string) error {
	return deleteByField(s.db, "name", name, &Provider{}, "provider")
}

type QueryProvider struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
	Type    string `json:"type"`
}

type ResQueryProvider struct {
	Page    Page       `json:"page"`
	Records []Provider `json:"records"`
}

// Page gets providers with pagination.
func (s *ProviderStorage) Page(query *QueryProvider) (*ResQueryProvider, error) {
	var res ResQueryProvider

	qry := s.db.Model(&Provider{})

	if query.KeyWord != "" {
		qry = qry.Where("name LIKE ? OR type LIKE ?", "%"+query.KeyWord+"%", "%"+query.KeyWord+"%")
	}

	if query.Type != "" {
		qry = qry.Where("type = ?", query.Type)
	}

	page, err := pageQuery(qry, "name", query.Page, &res.Records, "providers")
	if err != nil {
		return nil, err
	}
	res.Page = page

	return &res, nil
}

func (s *ProviderStorage) persistNormalizedProtocol(p *Provider) error {
	if p == nil {
		return fmt.Errorf("provider is nil")
	}

	original := p.Protocol
	if err := normalizeProviderProtocol(p); err != nil {
		return err
	}
	if original == p.Protocol {
		return nil
	}

	if err := s.db.Model(&Provider{}).Where("id = ?", p.ID).Update("protocol", p.Protocol).Error; err != nil {
		return fmt.Errorf("failed to persist provider protocol for %s: %w", p.Name, err)
	}
	return nil
}

func normalizeProviderProtocol(p *Provider) error {
	if p == nil {
		return fmt.Errorf("provider is nil")
	}

	if normalized := strings.ToLower(strings.TrimSpace(p.Protocol.ToString())); normalized != "" {
		p.Protocol = consts.ToProviderProtocol(normalized)
		return nil
	}

	protocol, ok := inferProviderProtocol(p)
	if !ok {
		return fmt.Errorf("provider protocol is required for type %s", p.Type)
	}
	p.Protocol = protocol
	return nil
}

func inferProviderProtocol(p *Provider) (consts.ProviderProtocol, bool) {
	switch p.Type {
	case consts.ProviderAnthropic:
		return consts.ProtocolAnthropic, true
	case consts.ProviderOpenAI,
		consts.ProviderDeepSeek,
		consts.ProviderOpenRouter,
		consts.ProviderGemini,
		consts.ProviderMistral,
		consts.ProviderGroq,
		consts.ProviderAzure,
		consts.ProviderOllama,
		consts.ProviderMoonshot,
		consts.ProviderZhipu,
		consts.ProviderQwen,
		consts.ProviderQwenCodingPlan,
		consts.ProviderSiliconFlow,
		consts.ProviderGrok:
		return consts.ProtocolOpenAI, true
	case consts.ProviderMiniMax:
		return inferMiniMaxProtocol(p), true
	default:
		return "", false
	}
}

func inferMiniMaxProtocol(p *Provider) consts.ProviderProtocol {
	if p == nil {
		return consts.ProtocolAnthropic
	}

	if protocol := readLegacyMiniMaxProtocol(p.Metadata); protocol != "" {
		return protocol
	}
	if strings.TrimSpace(p.Config) != "" {
		var raw map[string]any
		if err := json.Unmarshal([]byte(p.Config), &raw); err == nil {
			if protocol := readLegacyMiniMaxProtocol(raw); protocol != "" {
				return protocol
			}
		}
	}

	return consts.ProtocolAnthropic
}

func readLegacyMiniMaxProtocol(raw map[string]any) consts.ProviderProtocol {
	if raw == nil {
		return ""
	}

	value, ok := raw["api_format"].(string)
	if !ok {
		return ""
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case consts.ProtocolOpenAI.ToString():
		return consts.ProtocolOpenAI
	case consts.ProtocolAnthropic.ToString():
		return consts.ProtocolAnthropic
	default:
		return ""
	}
}
