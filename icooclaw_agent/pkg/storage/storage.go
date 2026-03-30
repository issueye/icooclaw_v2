// Package storage provides data storage for icooclaw using GORM.
package storage

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Storage provides SQLite-based storage using GORM.
type Storage struct {
	db        *gorm.DB
	path      string
	skill     *SkillStorage
	session   *SessionStorage
	message   MessageStore
	memory    *MemoryStorage
	tool      *ToolStorage
	agent     *AgentStorage
	provider  *ProviderStorage
	mcp       *MCPStorage
	channel   *ChannelStorage
	param     *ParamStorage
	execEnv   *ExecEnvStorage
	task      *TaskStorage
	workspace *WorkspaceStorage
}

func (s *Storage) Skill() *SkillStorage {
	return s.skill
}

func (s *Storage) Session() *SessionStorage {
	return s.session
}

func (s *Storage) Memory() *MemoryStorage {
	return s.memory
}

func (s *Storage) Tool() *ToolStorage {
	return s.tool
}

func (s *Storage) Agent() *AgentStorage {
	return s.agent
}

func (s *Storage) Provider() *ProviderStorage {
	return s.provider
}

func (s *Storage) MCP() *MCPStorage {
	return s.mcp
}

func (s *Storage) Channel() *ChannelStorage {
	return s.channel
}

func (s *Storage) Message() MessageStore {
	return s.message
}

func (s *Storage) Param() *ParamStorage {
	return s.param
}

func (s *Storage) ExecEnv() *ExecEnvStorage {
	return s.execEnv
}

func (s *Storage) Task() *TaskStorage {
	return s.task
}

func (s *Storage) Workspace() *WorkspaceStorage {
	return s.workspace
}

// New creates a new Storage instance.
func New(workspace string, mode string, path string) (*Storage, error) {
	return NewWithOptions(workspace, mode, path, MessageStoreOptions{Type: MessageStoreTypeSQLite})
}

// NewWithOptions creates a new Storage instance with message store options.
func NewWithOptions(workspace string, mode string, path string, messageOpts MessageStoreOptions) (*Storage, error) {
	db, err := gorm.Open(sqlite.Open(path+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 开启调试模式
	if mode == "debug" {
		db = db.Debug()
	}

	// Get underlying sql.DB 获取数据库连接池设置
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying db: %w", err)
	}
	sqlDB.SetMaxOpenConns(1) // SQLite recommends single connection
	sqlDB.SetMaxIdleConns(1)

	s := &Storage{
		db:        db,
		path:      path,
		skill:     NewSkillStorage(db),
		session:   NewSessionStorage(db),
		message:   NewMessageStorage(db),
		memory:    NewMemoryStorage(db),
		tool:      NewToolStorage(db),
		agent:     NewAgentStorage(db),
		provider:  NewProviderStorage(db),
		mcp:       NewMCPStorage(db),
		channel:   NewChannelStorage(db),
		param:     NewParamStorage(db),
		execEnv:   NewExecEnvStorage(db),
		task:      NewTaskStorage(db),
		workspace: NewWorkspaceStorage(workspace),
	}

	if err := s.autoMigrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}
	if err := s.provider.NormalizeProtocols(); err != nil {
		return nil, fmt.Errorf("failed to normalize providers: %w", err)
	}

	// 初始化技能
	if err := s.skill.ExistCreate(); err != nil {
		return nil, fmt.Errorf("failed to create skills: %w", err)
	}
	// 初始化渠道
	if err := s.channel.ExistCreate(); err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}
	// 初始化默认智能体
	if _, err := s.agent.GetDefault(); err != nil {
		return nil, fmt.Errorf("failed to ensure default agent: %w", err)
	}

	if err := s.initMessageStore(workspace, messageOpts); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Storage) initMessageStore(workspace string, opts MessageStoreOptions) error {
	storeType := strings.ToLower(strings.TrimSpace(opts.Type))
	if storeType == "" {
		storeType = MessageStoreTypeSQLite
	}

	switch storeType {
	case MessageStoreTypeSQLite:
		s.message = NewMessageStorage(s.db)
		return nil
	case MessageStoreTypeJSON:
		indexPath := strings.TrimSpace(opts.Path)
		if indexPath == "" {
			indexPath = filepath.Join(workspace, "messages", "messages.json")
		}
		store, err := NewJSONMessageStore(indexPath)
		if err != nil {
			return fmt.Errorf("init json message store failed: %w", err)
		}
		s.message = store
		return nil
	case MessageStoreTypeMarkdown:
		indexPath := strings.TrimSpace(opts.Path)
		if indexPath == "" {
			indexPath = filepath.Join(workspace, "messages", "messages_index.json")
		}
		dir := strings.TrimSpace(opts.Dir)
		if dir == "" {
			dir = filepath.Join(workspace, "messages", "markdown")
		}
		store, err := NewMarkdownMessageStore(indexPath, dir)
		if err != nil {
			return fmt.Errorf("init markdown message store failed: %w", err)
		}
		s.message = store
		return nil
	default:
		return fmt.Errorf("unsupported message store type: %s", storeType)
	}
}

// autoMigrate runs auto migration for all models.
func (s *Storage) autoMigrate() error {
	return s.db.AutoMigrate(
		&Provider{},
		&Channel{},
		&Session{},
		&Binding{},
		&Memory{},
		&Message{},
		&Tool{},
		&Agent{},
		&Skill{},
		&MCPConfig{},
		&ParamConfig{},
		&ExecEnv{},
		&Task{},
	)
}

// Close closes the database connection.
func (s *Storage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// DB returns the underlying GORM database.
func (s *Storage) DB() *gorm.DB {
	return s.db
}
