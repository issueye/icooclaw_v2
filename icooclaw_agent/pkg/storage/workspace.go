package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"icooclaw/pkg/utils"
)

const (
	WorkspacePromptAgents = "AGENTS"
	WorkspacePromptSoul   = "SOUL"
	WorkspacePromptUser   = "USER"
)

// WorkspaceStorage 工作空间存储
type WorkspaceStorage struct {
	workspace string
}

// NewWorkspaceStorage 创建工作空间存储
func NewWorkspaceStorage(workspace string) *WorkspaceStorage {
	return &WorkspaceStorage{workspace: workspace}
}

// GetWorkspace 获取工作空间
func (s *WorkspaceStorage) GetWorkspace() string {
	return s.workspace
}

// SetWorkspace 设置工作空间
func (s *WorkspaceStorage) SetWorkspace(workspace string) {
	s.workspace = workspace
}

func NormalizeWorkspacePromptName(name string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case WorkspacePromptAgents:
		return WorkspacePromptAgents, nil
	case WorkspacePromptSoul:
		return WorkspacePromptSoul, nil
	case WorkspacePromptUser:
		return WorkspacePromptUser, nil
	default:
		return "", fmt.Errorf("unsupported workspace prompt: %s", name)
	}
}

func (s *WorkspaceStorage) promptPath(name string) (string, error) {
	normalized, err := NormalizeWorkspacePromptName(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(s.workspace, normalized+".md"), nil
}

// AGENTS 工作空间下的智能体
func (s *WorkspaceStorage) Load(name string) (string, error) {
	path, err := s.promptPath(name)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func (s *WorkspaceStorage) Save(name, content string) error {
	path, err := s.promptPath(name)
	if err != nil {
		return err
	}
	return utils.WriteFileAtomic(path, []byte(content), 0o644)
}

// SOUL 人设配置文件
func (s *WorkspaceStorage) LoadSOUL() (string, error) {
	path := filepath.Join(s.workspace, "SOUL.md")

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// USER 用户配置文件
func (s *WorkspaceStorage) LoadUSER() (string, error) {
	path := filepath.Join(s.workspace, "USER.md")

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func (s *WorkspaceStorage) LoadWorkspace() (string, error) {
	soulPrompt, err := s.Load("SOUL")
	if err != nil {
		return "", err
	}

	userPrompt, err := s.Load("USER")
	if err != nil {
		return "", err
	}

	// 合并智能体、人设和用户配置
	sb := strings.Builder{}
	sb.WriteString("\n")
	sb.WriteString(soulPrompt)
	sb.WriteString("\n")
	sb.WriteString(userPrompt)
	sb.WriteString("\n")

	return sb.String(), nil
}
