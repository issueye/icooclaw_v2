// Package config 提供 icooclaw 的配置管理能力。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"icooclaw/pkg/consts"

	"github.com/spf13/viper"
)

const (
	defaultWorkspaceFileSoul = "SOUL.md"
	defaultWorkspaceFileUser = "USER.md"
	defaultHooksScriptPath   = "hooks/hooks.js"

	defaultWorkspaceDirSkills  = "skills"
	defaultWorkspaceDirTemp    = "temp"
	defaultWorkspaceDirReport  = "report"
	defaultWorkspaceDirReports = "reports"
	defaultWorkspaceDirScripts = "scripts"

	defaultModeRelease       = "release"
	defaultModeDebug         = "debug"
	defaultClawHubAddr       = "https://wry-manatee-359.convex.site"
	defaultWorkspacePath     = "./workspace"
	defaultLoggingPath       = "./workspace/logs/icooclaw.log"
	defaultDatabasePath      = "./workspace/data/icooclaw.db"
	defaultMessageStoreType  = "sqlite"
	defaultGatewayModeLocal  = "local"
	defaultLoggingLevelInfo  = "info"
	defaultLoggingFormatJSON = "json"
	defaultNetworkProxy      = "http://127.0.0.1:7897"

	defaultConfigPath = "config.toml"
	defaultConfigType = "toml"
	defaultEnvPrefix  = "ICOOCLAW"

	configKeyMode                  = "mode"
	configKeyClawHubAddr           = "claw_hub_addr"
	configKeyWorkspace             = "workspace"
	configKeyAgentRecentCount      = "agent.recent_count"
	configKeyDatabasePath          = "database.path"
	configKeyGatewayEnabled        = "gateway.enabled"
	configKeyGatewayPort           = "gateway.port"
	configKeyGatewaySecurityMode   = "gateway.security.mode"
	configKeyGatewaySecurityAPIKey = "gateway.security.api_key"
	configKeyLoggingLevel          = "logging.level"
	configKeyLoggingFormat         = "logging.format"
	configKeyLoggingPath           = "logging.path"
	configKeyLoggingAddSource      = "logging.add_source"
	configKeyExecTimeout           = "exec.timeout"
	configKeyExecEnv               = "exec.env"
	configKeyMessageStoreType      = "message_store.type"
	configKeyMessageStorePath      = "message_store.path"
	configKeyMessageStoreDir       = "message_store.dir"

	errWorkspaceRequired           = "workspace 是必需的"
	errWorkspaceMustBeAbs          = "workspace 必须是绝对路径"
	errDatabasePathRequired        = "database.path 是必需的"
	errGatewayPortRange            = "gateway.port 必须在 1 到 65535 之间"
	errGatewaySecurityModeRequired = "gateway.security.mode 是必需的"
	errGatewaySecurityModeInvalid  = "gateway.security.mode 必须是 local、apikey 或 open"
	errGatewayAPIKeyRequired       = "gateway.security.api_key 在 apikey 模式下是必需的"
	errModeInvalid                 = "mode 必须是 debug 或 release"
	errLoggingLevelInvalid         = "logging.level 必须是 debug、info、warn 或 error"
	errLoggingFormatInvalid        = "logging.format 必须是 json 或 text"
	errMessageStoreTypeInvalid     = "message_store.type 必须是 sqlite、json 或 markdown"
)

var (
	validSecurityModes = map[string]bool{defaultGatewayModeLocal: true, "apikey": true, "open": true}
	validModes         = map[string]bool{defaultModeDebug: true, defaultModeRelease: true}
	validLogLevels     = map[string]bool{defaultModeDebug: true, defaultLoggingLevelInfo: true, "warn": true, "error": true}
	validLogFormats    = map[string]bool{defaultLoggingFormatJSON: true, "text": true}
	validMessageStores = map[string]bool{"sqlite": true, "json": true, "markdown": true}
)

var defaultWorkspaceFiles = map[string]string{
	defaultWorkspaceFileSoul: `# Soul

## 角色定位
你是一个稳健、直接、可靠的 AI 助手。优先解决问题，而不是堆砌话术。

## 核心特质

- 忠诚可靠：优先维护用户利益、数据安全和结果正确性
- 清晰直接：表达简洁，结论明确，不绕弯
- 审慎执行：先判断边界和风险，再行动
- 善于分工：必要时会拆分任务并使用 subagent，但不会滥用委托

## 价值观

- 准确优先：先保证正确，再追求速度
- 透明公开：说明关键假设、风险和限制
- 最终负责：即使使用 subagent，主 agent 也要负责最终答案
- 持续改进：根据反馈修正流程和结果
`,
	defaultWorkspaceFileUser: `# User

## Preferences

- Communication style: concise
- Timezone: Asia/Shanghai
- Language: Chinese

## Notes

- Add stable user preferences here
- Do not store plaintext secrets in this file
`,
	defaultHooksScriptPath: `/**
 * Default hooks file.
 * All hooks are optional.
 */

function onGetProvider(defaultModel) {
    return defaultModel;
}
`,
}

var defaultWorkspaceDirs = []string{
	consts.HookScriptDir,
	defaultWorkspaceDirSkills,
	defaultWorkspaceDirTemp,
	defaultWorkspaceDirReport,
	defaultWorkspaceDirReports,
	defaultWorkspaceDirScripts,
}

// Config 表示应用配置结构。
type Config struct {
	Mode         string             `mapstructure:"mode"`          // 模式 debug 或 release
	ClawHubAddr  string             `mapstructure:"claw_hub_addr"` // 中心节点地址
	Workspace    string             `mapstructure:"workspace"`     // 工作空间
	Agent        AgentConfig        `mapstructure:"agent"`         // 基本智能体配置
	Database     DatabaseConfig     `mapstructure:"database"`      // 数据库配置
	Gateway      GatewayConfig      `mapstructure:"gateway"`       // 网关配置
	Logging      LoggingConfig      `mapstructure:"logging"`       // 日志配置
	Network      NetworkConfig      `mapstructure:"network"`       // 网络配置
	Exec         ExecConfig         `mapstructure:"exec"`          // 命令执行配置
	MessageStore MessageStoreConfig `mapstructure:"message_store"` // 消息存储配置
}

// AgentConfig 表示基础智能体配置。
type AgentConfig struct {
	RecentCount int `mapstructure:"recent_count"` // 最近消息数量，默认5
}

// DatabaseConfig 表示数据库配置。
type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

// GatewayConfig 表示网关配置。
type GatewayConfig struct {
	Enabled  bool                  `mapstructure:"enabled"`
	Port     int                   `mapstructure:"port"`
	Security GatewaySecurityConfig `mapstructure:"security"`
}

// GatewaySecurityConfig 表示网关安全配置。
type GatewaySecurityConfig struct {
	Mode   string `mapstructure:"mode"`
	APIKey string `mapstructure:"api_key"`
}

// LoggingConfig 表示日志配置。
type LoggingConfig struct {
	Level     string `mapstructure:"level"`
	Format    string `mapstructure:"format"`
	Path      string `mapstructure:"path"`
	AddSource bool   `mapstructure:"add_source"`
}

// NetworkConfig 表示网络配置。
type NetworkConfig struct {
	Proxy string `mapstructure:"proxy"` // 代理地址
}

type ExecConfig struct {
	Timeout int               `mapstructure:"timeout"`
	Env     map[string]string `mapstructure:"env"`
}

type MessageStoreConfig struct {
	Type string `mapstructure:"type"`
	Path string `mapstructure:"path"`
	Dir  string `mapstructure:"dir"`
}

// DefaultConfig 返回默认配置。
func DefaultConfig() *Config {
	return &Config{
		Mode:        defaultModeRelease,
		ClawHubAddr: defaultClawHubAddr,
		Workspace:   defaultWorkspacePath,
		Agent: AgentConfig{
			RecentCount: consts.DefaultRecentCount,
		},
		Database: DatabaseConfig{
			Path: defaultDatabasePath,
		},
		Gateway: GatewayConfig{
			Enabled: true,
			Port:    16789,
			Security: GatewaySecurityConfig{
				Mode: defaultGatewayModeLocal,
			},
		},
		Logging: LoggingConfig{
			Level:     defaultLoggingLevelInfo,
			Format:    defaultLoggingFormatJSON,
			Path:      defaultLoggingPath,
			AddSource: false,
		},
		Network: NetworkConfig{
			Proxy: defaultNetworkProxy,
		},
		Exec: ExecConfig{
			Timeout: 60,
			Env:     map[string]string{},
		},
		MessageStore: MessageStoreConfig{
			Type: defaultMessageStoreType,
			Path: "",
			Dir:  "",
		},
	}
}

// Load 从配置文件和环境变量加载配置。
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		path = defaultConfigPath
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := WriteDefaultConfig(path, cfg); err != nil {
			return nil, fmt.Errorf("创建默认配置文件失败: %w", err)
		}
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType(defaultConfigType)

	// 写入默认值
	setDefaults(v, cfg)

	// 启用环境变量覆盖
	v.SetEnvPrefix(defaultEnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 反序列化到配置结构体
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return cfg, nil
}

// WriteDefaultConfig 在配置文件不存在时写入默认配置文件。
func WriteDefaultConfig(path string, cfg *Config) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("创建配置目录失败: %w", err)
		}
	}

	content := fmt.Sprintf(`mode = %q
claw_hub_addr = %q
workspace = %q

[agent]
recent_count = %d

[database]
path = %q

[gateway]
enabled = %t
port = %d

[gateway.security]
mode = %q
api_key = %q

[logging]
level = %q
format = %q
path = %q
add_source = %t

[network]
proxy = %q

[exec]
timeout = %d

[exec.env]

[message_store]
type = %q
path = %q
dir = %q
`,
		cfg.Mode,
		cfg.ClawHubAddr,
		cfg.Workspace,
		cfg.Agent.RecentCount,
		cfg.Database.Path,
		cfg.Gateway.Enabled,
		cfg.Gateway.Port,
		cfg.Gateway.Security.Mode,
		cfg.Gateway.Security.APIKey,
		cfg.Logging.Level,
		cfg.Logging.Format,
		cfg.Logging.Path,
		cfg.Logging.AddSource,
		cfg.Network.Proxy,
		cfg.Exec.Timeout,
		cfg.MessageStore.Type,
		cfg.MessageStore.Path,
		cfg.MessageStore.Dir,
	)

	if len(cfg.Exec.Env) > 0 {
		keys := make([]string, 0, len(cfg.Exec.Env))
		for key := range cfg.Exec.Env {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			content += fmt.Sprintf("%s = %q\n", key, cfg.Exec.Env[key])
		}
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("写入默认配置文件失败: %w", err)
	}
	return nil
}

// setDefaults 向 viper 写入默认值。
func setDefaults(v *viper.Viper, cfg *Config) {
	v.SetDefault(configKeyMode, cfg.Mode)
	v.SetDefault(configKeyClawHubAddr, cfg.ClawHubAddr)
	v.SetDefault(configKeyWorkspace, cfg.Workspace)
	v.SetDefault(configKeyAgentRecentCount, cfg.Agent.RecentCount)
	v.SetDefault(configKeyDatabasePath, cfg.Database.Path)
	v.SetDefault(configKeyGatewayEnabled, cfg.Gateway.Enabled)
	v.SetDefault(configKeyGatewayPort, cfg.Gateway.Port)
	v.SetDefault(configKeyGatewaySecurityMode, cfg.Gateway.Security.Mode)
	v.SetDefault(configKeyGatewaySecurityAPIKey, cfg.Gateway.Security.APIKey)
	v.SetDefault(configKeyLoggingLevel, cfg.Logging.Level)
	v.SetDefault(configKeyLoggingFormat, cfg.Logging.Format)
	v.SetDefault(configKeyLoggingPath, cfg.Logging.Path)
	v.SetDefault(configKeyLoggingAddSource, cfg.Logging.AddSource)
	v.SetDefault(configKeyExecTimeout, cfg.Exec.Timeout)
	v.SetDefault(configKeyExecEnv, cfg.Exec.Env)
	v.SetDefault(configKeyMessageStoreType, cfg.MessageStore.Type)
	v.SetDefault(configKeyMessageStorePath, cfg.MessageStore.Path)
	v.SetDefault(configKeyMessageStoreDir, cfg.MessageStore.Dir)
}

// Validate 校验配置是否有效。
func (c *Config) Validate() error {
	var errs []error

	if c.Workspace == "" {
		errs = append(errs, fmt.Errorf(errWorkspaceRequired))
	}

	if c.Database.Path == "" {
		errs = append(errs, fmt.Errorf(errDatabasePathRequired))
	}

	if c.Gateway.Enabled {
		if c.Gateway.Port <= 0 || c.Gateway.Port > 65535 {
			errs = append(errs, fmt.Errorf(errGatewayPortRange))
		}

		if c.Gateway.Security.Mode == "" {
			errs = append(errs, fmt.Errorf(errGatewaySecurityModeRequired))
		} else if !validSecurityModes[c.Gateway.Security.Mode] {
			errs = append(errs, fmt.Errorf(errGatewaySecurityModeInvalid))
		}

		if c.Gateway.Security.Mode == "apikey" && strings.TrimSpace(c.Gateway.Security.APIKey) == "" {
			errs = append(errs, fmt.Errorf(errGatewayAPIKeyRequired))
		}
	}

	if c.Mode != "" && !validModes[c.Mode] {
		errs = append(errs, fmt.Errorf(errModeInvalid))
	}

	if c.Logging.Level != "" && !validLogLevels[c.Logging.Level] {
		errs = append(errs, fmt.Errorf(errLoggingLevelInvalid))
	}

	if c.Logging.Format != "" && !validLogFormats[c.Logging.Format] {
		errs = append(errs, fmt.Errorf(errLoggingFormatInvalid))
	}
	if c.MessageStore.Type != "" && !validMessageStores[c.MessageStore.Type] {
		errs = append(errs, fmt.Errorf(errMessageStoreTypeInvalid))
	}

	if len(errs) > 0 {
		return fmt.Errorf("配置验证失败: %v", errs)
	}

	return nil
}

type ValidationError struct {
	Field   string
	Message string
}

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", ve.Field, ve.Message)
}

func (c *Config) ValidateStrict() []*ValidationError {
	var errors []*ValidationError

	if c.Workspace == "" {
		errors = append(errors, &ValidationError{Field: configKeyWorkspace, Message: errWorkspaceRequired})
	} else if !filepath.IsAbs(c.Workspace) {
		errors = append(errors, &ValidationError{Field: configKeyWorkspace, Message: errWorkspaceMustBeAbs})
	}

	if c.Database.Path == "" {
		errors = append(errors, &ValidationError{Field: configKeyDatabasePath, Message: errDatabasePathRequired})
	}

	if c.Gateway.Enabled {
		if c.Gateway.Port <= 0 || c.Gateway.Port > 65535 {
			errors = append(errors, &ValidationError{Field: configKeyGatewayPort, Message: errGatewayPortRange})
		}

		if c.Gateway.Security.Mode == "" {
			errors = append(errors, &ValidationError{Field: configKeyGatewaySecurityMode, Message: errGatewaySecurityModeRequired})
		} else if !validSecurityModes[c.Gateway.Security.Mode] {
			errors = append(errors, &ValidationError{Field: configKeyGatewaySecurityMode, Message: errGatewaySecurityModeInvalid})
		}

		if c.Gateway.Security.Mode == "apikey" && strings.TrimSpace(c.Gateway.Security.APIKey) == "" {
			errors = append(errors, &ValidationError{Field: configKeyGatewaySecurityAPIKey, Message: errGatewayAPIKeyRequired})
		}
	}

	if c.Mode != "" && !validModes[c.Mode] {
		errors = append(errors, &ValidationError{Field: configKeyMode, Message: errModeInvalid})
	}

	if c.Logging.Level != "" && !validLogLevels[c.Logging.Level] {
		errors = append(errors, &ValidationError{Field: configKeyLoggingLevel, Message: errLoggingLevelInvalid})
	}

	if c.Logging.Format != "" && !validLogFormats[c.Logging.Format] {
		errors = append(errors, &ValidationError{Field: configKeyLoggingFormat, Message: errLoggingFormatInvalid})
	}
	if c.MessageStore.Type != "" && !validMessageStores[c.MessageStore.Type] {
		errors = append(errors, &ValidationError{Field: configKeyMessageStoreType, Message: errMessageStoreTypeInvalid})
	}

	return errors
}

// EnsureWorkspace 确保工作空间目录存在。
func (c *Config) EnsureWorkspace() error {
	if err := os.MkdirAll(c.Workspace, 0755); err != nil {
		return fmt.Errorf("创建工作目录失败: %w", err)
	}
	return nil
}

// EnsureWorkspaceScaffold 确保基础工作空间结构和种子文件存在。
func (c *Config) EnsureWorkspaceScaffold() error {
	if err := c.EnsureWorkspace(); err != nil {
		return err
	}

	for _, dir := range defaultWorkspaceDirs {
		if err := os.MkdirAll(filepath.Join(c.Workspace, dir), 0o755); err != nil {
			return fmt.Errorf("创建工作区目录失败 %s: %w", dir, err)
		}
	}

	for relPath, content := range defaultWorkspaceFiles {
		fullPath := filepath.Join(c.Workspace, relPath)
		if _, err := os.Stat(fullPath); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("检查工作区文件失败 %s: %w", relPath, err)
		}

		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return fmt.Errorf("创建工作区文件目录失败 %s: %w", relPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("写入工作区文件失败 %s: %w", relPath, err)
		}
	}

	return nil
}

// EnsureDatabasePath 确保数据库目录存在。
func (c *Config) EnsureDatabasePath() error {
	dir := filepath.Dir(c.Database.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建数据库目录失败:%w", err)
	}
	return nil
}

// GetWorkspacePath 返回工作空间的绝对路径。
func (c *Config) GetWorkspacePath() (string, error) {
	return filepath.Abs(c.Workspace)
}

// GetDatabasePath 返回数据库的绝对路径。
func (c *Config) GetDatabasePath() (string, error) {
	return filepath.Abs(c.Database.Path)
}
