package storage

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	icooclawErrors "icooclaw/pkg/errors"
	"icooclaw/pkg/utils"
)

type SkillType string

const (
	SkillTypeSkill  SkillType = "builtin" // 内置技能
	SkillTypeCustom SkillType = "custom"  // 自定义技能
)

func (s SkillType) String() string {
	return string(s)
}

// Skill represents a skill configuration.
type Skill struct {
	Model
	Name        string    `gorm:"column:name;type:varchar(100);not null;comment:技能名称" json:"name"`          // 技能名称
	Description string    `gorm:"column:description;type:text;comment:技能描述" json:"description"`             // 技能描述
	Title       string    `gorm:"column:title;type:text;comment:技能标题" json:"title"`                         // 技能标题
	Enabled     bool      `gorm:"column:enabled;type:tinyint(1);default:true;comment:是否启用" json:"enabled"`  // 是否启用
	Version     string    `gorm:"column:version;type:varchar(10);default:1.0.0;comment:版本号" json:"version"` // 版本号
	Type        SkillType `gorm:"column:type;type:varchar(10);default:skill;comment:技能类型" json:"type"`      // 技能类型
	Path        string    `gorm:"column:path;type:text;comment:技能路径" json:"path"`                           // 技能路径 默认 workspace/.skills/<name>-<version>/
}

// TableName returns the table name for Skill.
func (Skill) TableName() string {
	return tableNamePrefix + "skills"
}

type SkillStorage struct {
	db *gorm.DB
}

func NewSkillStorage(db *gorm.DB) *SkillStorage {
	return &SkillStorage{db: db}
}

// SaveSkill saves a skill configuration (creates or updates based on name).
func (s *SkillStorage) SaveSkill(sk *Skill) error {
	if sk == nil {
		return fmt.Errorf("skill is nil")
	}

	existing, err := s.findSkillByIdentifier(sk.Name)
	if err != nil && err != icooclawErrors.ErrRecordNotFound {
		return fmt.Errorf("failed to save skill: %w", err)
	}

	if existing != nil {
		sk.ID = existing.ID
		return s.db.Model(&Skill{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"name":        sk.Name,
			"description": sk.Description,
			"title":       sk.Title,
			"enabled":     sk.Enabled,
			"version":     sk.Version,
			"type":        sk.Type,
			"path":        sk.Path,
		}).Error
	}

	return s.db.Create(sk).Error
}

// GetSkill gets a skill by name.
func (s *SkillStorage) GetSkill(name string) (*Skill, error) {
	sk, err := s.findSkillByIdentifier(name)
	if err != nil {
		if err == icooclawErrors.ErrRecordNotFound {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get skill: %w", err)
	}
	return sk, nil
}

func (s *SkillStorage) GetSkillByID(id string) (*Skill, error) {
	sk := Skill{}
	result := s.db.Where("id = ?", id).First(&sk)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, icooclawErrors.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get skill by id: %w", result.Error)
	}
	return &sk, nil
}

// ListSkills lists all skills.
func (s *SkillStorage) ListSkills() ([]*Skill, error) {
	var skills []*Skill
	result := s.db.Order("name").Find(&skills)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list skills: %w", result.Error)
	}
	return skills, nil
}

// ListEnabledSkills lists all enabled skills.
func (s *SkillStorage) ListEnabledSkills() ([]*Skill, error) {
	var skills []*Skill
	result := s.db.Where("enabled = ?", true).Order("name").Find(&skills)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list enabled skills: %w", result.Error)
	}
	return skills, nil
}

// DeleteSkill deletes a skill by id.
func (s *SkillStorage) DeleteSkill(id string) error {
	result := s.db.Where("id = ?", id).Delete(&Skill{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete skill: %w", result.Error)
	}
	return nil
}

func (s *SkillStorage) DeleteSkillByName(name string) error {
	skill, err := s.GetSkill(name)
	if err != nil {
		return err
	}
	return s.DeleteSkill(skill.ID)
}

type QuerySkill struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
	Enabled *bool  `json:"enabled"`
}

type ResQuerySkill struct {
	Page    Page    `json:"page"`
	Records []Skill `json:"records"`
}

// Page gets skills with pagination.
func (s *SkillStorage) Page(query *QuerySkill) (*ResQuerySkill, error) {
	var res ResQuerySkill

	qry := s.db.Model(&Skill{})

	if query.KeyWord != "" {
		qry = qry.Where("name LIKE ? OR description LIKE ?", "%"+query.KeyWord+"%", "%"+query.KeyWord+"%")
	}

	if query.Enabled != nil {
		qry = qry.Where("enabled = ?", *query.Enabled)
	}

	qry = qry.Order("name")

	result := qry.Count(&res.Page.Total)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to count skills: %w", result.Error)
	}

	if query.Page.Page == 0 || query.Page.Size == 0 {
		result = qry.Find(&res.Records)
	} else {
		result = qry.Limit(query.Page.Size).
			Offset((query.Page.Page - 1) * query.Page.Size).
			Find(&res.Records)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get skills: %w", result.Error)
	}

	return &res, nil
}

func (s *SkillStorage) ExistCreate() error {
	skills := []*Skill{
		{
			Name:        "amap-weather",
			Description: `Query weather information using AMap (高德地图) Weather API. Supports real-time weather and 4-day forecasts for any city in China.`,
			Title:       "AMap Weather",
			Version:     "1.0.0",
			Enabled:     true,
			Type:        SkillTypeSkill,
			Path:        filepathJoin("skills", "amap-weather-1.0.0"),
		},
		{
			Name:        "skill-creator",
			Description: "Create a new skill",
			Title:       "Skill Creator",
			Version:     "1.0.0",
			Enabled:     true,
			Type:        SkillTypeSkill,
			Path:        filepathJoin("skills", "skill-creator-1.0.0"),
		},
		{
			Name:        "weather",
			Description: "Query weather information",
			Title:       "Weather",
			Version:     "",
			Enabled:     true,
			Type:        SkillTypeSkill,
			Path:        filepathJoin("skills", "weather"),
		},
		{
			Name:        "ip-lookup",
			Description: "Query IP address information",
			Title:       "IP Lookup",
			Version:     "1.0.0",
			Enabled:     true,
			Type:        SkillTypeSkill,
			Path:        filepathJoin("skills", "ip-lookup-1.0.0"),
		},
		{
			Name:        "chinese-calendar",
			Description: "Query Chinese calendar information",
			Title:       "Chinese Calendar",
			Version:     "1.0.0",
			Enabled:     true,
			Type:        SkillTypeSkill,
			Path:        filepathJoin("skills", "chinese-calendar-1.0.0"),
		},
		{
			Name:        "baidu-search",
			Description: "Search information using Baidu AI Search API",
			Title:       "Baidu Search",
			Version:     "1.0.0",
			Enabled:     true,
			Type:        SkillTypeSkill,
			Path:        filepathJoin("skills", "baidu-search-1.0.0"),
		},
		{
			Name:        "tophub",
			Description: "Query trending topics from TopHub",
			Title:       "TopHub Hot List",
			Version:     "1.0.0",
			Enabled:     true,
			Type:        SkillTypeSkill,
			Path:        filepathJoin("skills", "tophub-1.0.0"),
		},
		{
			Name:        "tencent-docs",
			Description: "Query information from Tencent Docs",
			Title:       "Tencent Docs",
			Version:     "1.0.19",
			Enabled:     true,
			Type:        SkillTypeSkill,
			Path:        filepathJoin("skills", "tencent-docs-1.0.19"),
		},
	}

	for _, skill := range skills {
		if err := s.SaveSkill(skill); err != nil {
			return fmt.Errorf("failed to save skill %s: %w", skill.Name, err)
		}
	}

	return nil
}

func (s *SkillStorage) findSkillByIdentifier(identifier string) (*Skill, error) {
	trimmed := strings.TrimSpace(identifier)
	if trimmed == "" {
		return nil, icooclawErrors.ErrRecordNotFound
	}

	sk := Skill{}
	result := s.db.Where("name = ? OR title = ?", trimmed, trimmed).First(&sk)
	if result.Error == nil {
		return &sk, nil
	}
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return nil, result.Error
	}

	normalized := utils.NormalizeSkillIdentifier(trimmed)
	if normalized == "" {
		return nil, icooclawErrors.ErrRecordNotFound
	}

	var skills []*Skill
	if err := s.db.Find(&skills).Error; err != nil {
		return nil, err
	}

	for _, candidate := range skills {
		if utils.NormalizeSkillIdentifier(candidate.Name) == normalized ||
			utils.NormalizeSkillIdentifier(candidate.Title) == normalized {
			return candidate, nil
		}
	}

	return nil, icooclawErrors.ErrRecordNotFound
}

func filepathJoin(parts ...string) string {
	return strings.Join(parts, string('/'))
}
