// Package skill provides skill management for icooclaw.
package skill

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"icooclaw/pkg/storage"
)

// Skill represents a skill that can be activated.
type Metadata struct {
	Name        string `json:"name"`        // 技能名称
	Title       string `json:"title"`       // 技能显示名称
	Description string `json:"description"` // 技能描述
	Version     string `json:"version"`     // 技能版本
}

type Info struct {
	Metadata
	// Path is the path to the skill file.
	Path    string `json:"path"`    // 技能路径
	Source  string `json:"source"`  // 技能来源
	Content string `json:"content"` // 技能内容
}

func (info Info) validate() error {
	var errs error
	if info.Name == "" {
		errs = errors.Join(errs, errors.New("name is required"))
	} else {
		if len(info.Name) > MaxNameLength {
			errs = errors.Join(errs, fmt.Errorf("name exceeds %d characters", MaxNameLength))
		}
		if !namePattern.MatchString(info.Name) {
			errs = errors.Join(errs, errors.New("name must be alphanumeric with hyphens"))
		}
	}

	if info.Description == "" {
		errs = errors.Join(errs, errors.New("description is required"))
	} else if len(info.Description) > MaxDescriptionLength {
		errs = errors.Join(errs, fmt.Errorf("description exceeds %d character", MaxDescriptionLength))
	}
	return errs
}

type SkillsLoader struct {
	workspace       string
	workspaceSkills string // workspace skills (project-level)
	builtinSkills   string // builtin skills
}

// Loader 加载技能接口。
type Loader interface {
	LoadMetadata(ctx context.Context, name string) (*Metadata, error) // 加载元数据
	LoadInfo(ctx context.Context, name string) (*Info, error)         // 加载详细信息
	List(ctx context.Context) ([]*Info, error)                        // 列出所有技能
}

// DefaultLoader 默认技能加载器。
type DefaultLoader struct {
	workspace string
	storage   *storage.Storage
	logger    *slog.Logger
	mu        sync.RWMutex
}

// NewLoader 创建一个新的技能加载器。
func NewLoader(workspace string, s *storage.Storage, logger *slog.Logger) *DefaultLoader {
	if logger == nil {
		logger = slog.Default()
	}
	return &DefaultLoader{
		workspace: workspace,
		storage:   s,
		logger:    logger,
	}
}

// LoadMetadata 加载技能元数据
func (l *DefaultLoader) LoadMetadata(ctx context.Context, name string) (*Metadata, error) {
	go func() {
		select {
		case <-ctx.Done():
			return
		default:
		}
	}()

	// 从存储中获取技能信息
	sk, err := l.storage.Skill().GetSkill(name)
	if err != nil {
		return nil, fmt.Errorf("skill %s not found: %w", name, err)
	}

	// Parse skill
	skill := &Metadata{
		Name:        sk.Name,
		Title:       sk.Title,
		Description: sk.Description,
		Version:     sk.Version,
	}

	return skill, nil
}

// LoadInfo 加载技能详细信息
func (l *DefaultLoader) LoadInfo(ctx context.Context, name string) (*Info, error) {
	// 加载技能详细信息
	info, err := l.ReadSkill(ctx, name, "")
	if err != nil {
		return nil, fmt.Errorf("read skill %s info failed: %w", name, err)
	}

	return info, nil
}

// List 列出所有技能
func (l *DefaultLoader) List(ctx context.Context) ([]*Info, error) {
	// Check context cancellation
	go func() {
		select {
		case <-ctx.Done():
			return
		default:
		}
	}()

	skills, err := l.storage.Skill().ListEnabledSkills()
	if err != nil {
		return nil, err
	}

	result := make([]*Info, 0, len(skills))
	for _, sk := range skills {
		skill := &Info{
			Metadata: Metadata{
				Name:        sk.Name,
				Title:       sk.Title,
				Description: sk.Description,
				Version:     sk.Version,
			},
			Path: sk.Path,
		}

		result = append(result, skill)
	}

	return result, nil
}

// ReadSkill 从工作区中读取技能文件。
func (l *DefaultLoader) ReadSkill(ctx context.Context, name string, version string) (*Info, error) {
	_ = ctx

	sk, err := l.storage.Skill().GetSkill(name)
	if err != nil {
		return nil, fmt.Errorf("get skill %s failed: %w", name, err)
	}

	path, err := l.resolveSkillFilePath(sk, version)
	if err != nil {
		return nil, err
	}

	parseInfo, err := NewParser().ParseFile(path)
	if err != nil {
		return nil, fmt.Errorf("parse skill file %s failed: %w", path, err)
	}

	info := &Info{
		Metadata: Metadata{
			Name:        parseInfo.Name,
			Title:       firstNonEmpty(parseInfo.Title, sk.Title),
			Description: parseInfo.Description,
			Version:     parseInfo.Version,
		},
		Content: parseInfo.Content,
		Path:    path,
		Source:  firstNonEmpty(sk.Path, l.workspace),
	}

	return info, nil
}

func (l *DefaultLoader) resolveSkillFilePath(sk *storage.Skill, version string) (string, error) {
	candidates := make([]string, 0, 6)

	appendCandidate := func(path string) {
		if path == "" {
			return
		}
		candidates = append(candidates, path)
		if !filepath.IsAbs(path) {
			candidates = append(candidates, filepath.Join(l.workspace, path))
		}
	}

	appendCandidate(sk.Path)
	if version != "" {
		appendCandidate(filepath.Join("skills", fmt.Sprintf("%s-%s", sk.Name, version)))
	}
	if sk.Version != "" {
		appendCandidate(filepath.Join("skills", fmt.Sprintf("%s-%s", sk.Name, sk.Version)))
	}
	appendCandidate(filepath.Join("skills", sk.Name))

	for _, candidate := range candidates {
		resolved, err := normalizeSkillPath(candidate)
		if err != nil {
			continue
		}
		if _, err := os.Stat(resolved); err == nil {
			return resolved, nil
		}
	}

	return "", fmt.Errorf("skill file not found for %s", sk.Name)
}

func normalizeSkillPath(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) && filepath.Ext(path) == "" {
			skillFile := filepath.Join(path, "SKILL.md")
			if _, statErr := os.Stat(skillFile); statErr == nil {
				return skillFile, nil
			}
		}
		return "", err
	}

	if info.IsDir() {
		return filepath.Join(path, "SKILL.md"), nil
	}

	return path, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// Manager 管理技能注册和执行。
type Manager struct {
	loader  Loader
	storage *storage.Storage
	logger  *slog.Logger
	mu      sync.RWMutex
}
