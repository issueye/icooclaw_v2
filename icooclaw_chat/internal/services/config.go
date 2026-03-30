package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/pelletier/go-toml/v2"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ClawConnectionConfig struct {
	APIBase string `json:"apiBase" toml:"api_base"`
	WSHost  string `json:"wsHost" toml:"ws_host"`
	WSPort  string `json:"wsPort" toml:"ws_port"`
	WSPath  string `json:"wsPath" toml:"ws_path"`
	UserID  string `json:"userId" toml:"user_id"`
}

type AgentProcessConfig struct {
	BinaryPath string `json:"binaryPath" toml:"binary_path"`
}

type Config struct {
	ClawConnection ClawConnectionConfig `json:"clawConnection" toml:"claw_connection"`
	AgentProcess   AgentProcessConfig   `json:"agentProcess" toml:"agent_process"`
}

type ConfigService struct {
	config     *Config
	configPath string
	mu         sync.RWMutex
	ctx        context.Context
}

var configService *ConfigService
var configOnce sync.Once

func defaultConfig() *Config {
	return &Config{
		ClawConnection: ClawConnectionConfig{
			APIBase: "http://localhost:16789",
			WSHost:  "localhost",
			WSPort:  "16789",
			WSPath:  "/ws",
			UserID:  "user-1",
		},
		AgentProcess: AgentProcessConfig{
			BinaryPath: "",
		},
	}
}

func GetConfigService() *ConfigService {
	configOnce.Do(func() {
		configService = &ConfigService{
			config: defaultConfig(),
		}
	})
	return configService
}

func (s *ConfigService) Init(ctx context.Context) {
	s.ctx = ctx
	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = "."
	}
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		runtime.LogWarning(s.ctx, "Failed to create config directory: "+err.Error())
	}
	s.configPath = filepath.Join(workingDir, "icoo_chat.toml")
	if err := s.ensureConfigFile(); err != nil {
		runtime.LogWarning(s.ctx, "Failed to create default config file: "+err.Error())
		return
	}
	if err := s.Load(); err != nil {
		runtime.LogWarning(s.ctx, "Failed to load config file: "+err.Error())
	}
}

func (s *ConfigService) ensureConfigFile() error {
	if s.configPath == "" {
		return fmt.Errorf("config path is empty")
	}

	_, err := os.Stat(s.configPath)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}

	return s.Save()
}

func (s *ConfigService) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return err
	}

	cfg := defaultConfig()
	if err := toml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	s.applyDefaultsLocked(cfg)
	return nil
}

func (s *ConfigService) Save() error {
	s.mu.RLock()
	cfg := *s.config
	configPath := s.configPath
	s.mu.RUnlock()

	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func (s *ConfigService) GetConfig() *Config {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg := *s.config
	return &cfg
}

func (s *ConfigService) GetClawConnectionConfig() ClawConnectionConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.ClawConnection
}

func (s *ConfigService) SetClawConnectionConfig(cfg ClawConnectionConfig) error {
	s.mu.Lock()
	current := *s.config
	current.ClawConnection = cfg
	s.applyDefaultsLocked(&current)
	s.mu.Unlock()
	return s.Save()
}

func (s *ConfigService) GetAgentProcessConfig() AgentProcessConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.AgentProcess
}

func (s *ConfigService) SetAgentProcessConfig(cfg AgentProcessConfig) error {
	s.mu.Lock()
	current := *s.config
	current.AgentProcess = cfg
	s.applyDefaultsLocked(&current)
	s.mu.Unlock()
	return s.Save()
}

func (s *ConfigService) GetConfigJSON() string {
	s.mu.RLock()
	cfg := *s.config
	s.mu.RUnlock()

	data, _ := json.Marshal(cfg)
	return string(data)
}

func (s *ConfigService) applyDefaultsLocked(cfg *Config) {
	defaults := defaultConfig()
	if cfg.ClawConnection.APIBase == "" {
		cfg.ClawConnection.APIBase = defaults.ClawConnection.APIBase
	}
	if cfg.ClawConnection.WSHost == "" {
		cfg.ClawConnection.WSHost = defaults.ClawConnection.WSHost
	}
	if cfg.ClawConnection.WSPort == "" {
		cfg.ClawConnection.WSPort = defaults.ClawConnection.WSPort
	}
	if cfg.ClawConnection.WSPath == "" {
		cfg.ClawConnection.WSPath = defaults.ClawConnection.WSPath
	}
	if cfg.ClawConnection.UserID == "" {
		cfg.ClawConnection.UserID = defaults.ClawConnection.UserID
	}
	s.config = cfg
}
