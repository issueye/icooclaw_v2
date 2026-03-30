package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid config",
			config: &Config{
				Mode:      "debug",
				Workspace: "/valid/workspace",
				Database:  DatabaseConfig{Path: "/valid/db/path.db"},
				Gateway: GatewayConfig{
					Enabled: true,
					Port:    16789,
					Security: GatewaySecurityConfig{
						Mode: "local",
					},
				},
				Agent:   AgentConfig{RecentCount: 5},
				Logging: LoggingConfig{Level: "info", Format: "json"},
			},
			wantErr: false,
		},
		{
			name: "Empty workspace",
			config: &Config{
				Workspace: "",
				Database:  DatabaseConfig{Path: "/valid/db"},
			},
			wantErr: true,
			errMsg:  "workspace",
		},
		{
			name: "Empty database path",
			config: &Config{
				Workspace: "/valid/workspace",
				Database:  DatabaseConfig{Path: ""},
			},
			wantErr: true,
			errMsg:  "database.path",
		},
		{
			name: "Invalid gateway port - zero",
			config: &Config{
				Workspace: "/valid/workspace",
				Database:  DatabaseConfig{Path: "/valid/db"},
				Gateway:   GatewayConfig{Enabled: true, Port: 0, Security: GatewaySecurityConfig{Mode: "local"}},
			},
			wantErr: true,
			errMsg:  "gateway.port",
		},
		{
			name: "Invalid gateway port - too high",
			config: &Config{
				Workspace: "/valid/workspace",
				Database:  DatabaseConfig{Path: "/valid/db"},
				Gateway:   GatewayConfig{Enabled: true, Port: 70000, Security: GatewaySecurityConfig{Mode: "local"}},
			},
			wantErr: true,
			errMsg:  "gateway.port",
		},
		{
			name: "Invalid gateway security mode",
			config: &Config{
				Workspace: "/valid/workspace",
				Database:  DatabaseConfig{Path: "/valid/db"},
				Gateway:   GatewayConfig{Enabled: true, Port: 16789, Security: GatewaySecurityConfig{Mode: "invalid"}},
			},
			wantErr: true,
			errMsg:  "gateway.security.mode",
		},
		{
			name: "Missing gateway api key in apikey mode",
			config: &Config{
				Workspace: "/valid/workspace",
				Database:  DatabaseConfig{Path: "/valid/db"},
				Gateway:   GatewayConfig{Enabled: true, Port: 16789, Security: GatewaySecurityConfig{Mode: "apikey"}},
			},
			wantErr: true,
			errMsg:  "gateway.security.api_key",
		},
		{
			name: "Invalid mode",
			config: &Config{
				Mode:      "invalid",
				Workspace: "/valid/workspace",
				Database:  DatabaseConfig{Path: "/valid/db"},
			},
			wantErr: true,
			errMsg:  "mode",
		},
		{
			name: "Invalid log level",
			config: &Config{
				Workspace: "/valid/workspace",
				Database:  DatabaseConfig{Path: "/valid/db"},
				Logging:   LoggingConfig{Level: "invalid"},
			},
			wantErr: true,
			errMsg:  "logging.level",
		},
		{
			name: "Invalid log format",
			config: &Config{
				Workspace: "/valid/workspace",
				Database:  DatabaseConfig{Path: "/valid/db"},
				Logging:   LoggingConfig{Level: "info", Format: "invalid"},
			},
			wantErr: true,
			errMsg:  "logging.format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestConfig_ValidateStrict(t *testing.T) {
	workspaceAbs := filepath.Join(t.TempDir(), "workspace")
	dbAbs := filepath.Join(t.TempDir(), "data", "app.db")

	tests := []struct {
		name          string
		config        *Config
		expectedCount int
	}{
		{
			name: "Valid strict config",
			config: &Config{
				Mode:      "release",
				Workspace: workspaceAbs,
				Database:  DatabaseConfig{Path: dbAbs},
				Gateway:   GatewayConfig{Enabled: true, Port: 16789, Security: GatewaySecurityConfig{Mode: "local"}},
				Agent:     AgentConfig{RecentCount: 5},
				Logging:   LoggingConfig{Level: "info", Format: "json"},
			},
			expectedCount: 0,
		},
		{
			name: "Relative workspace path",
			config: &Config{
				Workspace: "./relative/workspace",
				Database:  DatabaseConfig{Path: dbAbs},
				Gateway:   GatewayConfig{Enabled: true, Port: 16789, Security: GatewaySecurityConfig{Mode: "local"}},
				Agent:     AgentConfig{RecentCount: 5},
				Logging:   LoggingConfig{Level: "info", Format: "json"},
			},
			expectedCount: 1,
		},
		{
			name: "Multiple errors",
			config: &Config{
				Workspace: "",
				Database:  DatabaseConfig{Path: ""},
				Gateway:   GatewayConfig{Enabled: true, Port: -1, Security: GatewaySecurityConfig{Mode: ""}},
				Agent:     AgentConfig{RecentCount: 5},
				Logging:   LoggingConfig{Level: "info", Format: "json"},
			},
			expectedCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.config.ValidateStrict()
			if len(errors) != tt.expectedCount {
				t.Errorf("ValidateStrict() returned %d errors, want %d", len(errors), tt.expectedCount)
			}
		})
	}
}

func TestConfig_DefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Mode != "release" {
		t.Errorf("DefaultConfig().Mode = %q, want release", cfg.Mode)
	}
	if cfg.Workspace != "./workspace" {
		t.Errorf("DefaultConfig().Workspace = %q, want ./workspace", cfg.Workspace)
	}
	if cfg.Database.Path != "./workspace/data/icooclaw.db" {
		t.Errorf("DefaultConfig().Database.Path = %q, want ./workspace/data/icooclaw.db", cfg.Database.Path)
	}
	if cfg.Gateway.Port != 16789 {
		t.Errorf("DefaultConfig().Gateway.Port = %d, want 16789", cfg.Gateway.Port)
	}
	if cfg.Agent.RecentCount != 5 {
		t.Errorf("DefaultConfig().Agent.RecentCount = %d, want 5", cfg.Agent.RecentCount)
	}
	if cfg.Gateway.Security.Mode != "local" {
		t.Errorf("DefaultConfig().Gateway.Security.Mode = %q, want local", cfg.Gateway.Security.Mode)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("DefaultConfig().Logging.Level = %q, want info", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("DefaultConfig().Logging.Format = %q, want json", cfg.Logging.Format)
	}
	if cfg.Logging.Path != "./workspace/logs/icooclaw.log" {
		t.Errorf("DefaultConfig().Logging.Path = %q, want ./workspace/logs/icooclaw.log", cfg.Logging.Path)
	}
	if cfg.Exec.Timeout != 60 {
		t.Errorf("DefaultConfig().Exec.Timeout = %d, want 60", cfg.Exec.Timeout)
	}
	if len(cfg.Exec.Env) != 0 {
		t.Errorf("DefaultConfig().Exec.Env = %#v, want empty", cfg.Exec.Env)
	}
}

func TestConfig_EnsureDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		Workspace: filepath.Join(tmpDir, "workspace"),
		Database:  DatabaseConfig{Path: filepath.Join(tmpDir, "data", "app.db")},
	}

	if err := cfg.EnsureWorkspace(); err != nil {
		t.Errorf("EnsureWorkspace() error = %v", err)
	}

	if err := cfg.EnsureDatabasePath(); err != nil {
		t.Errorf("EnsureDatabasePath() error = %v", err)
	}

	if _, err := os.Stat(cfg.Workspace); err != nil {
		t.Errorf("workspace directory not created: %v", err)
	}

	if _, err := os.Stat(filepath.Dir(cfg.Database.Path)); err != nil {
		t.Errorf("database directory not created: %v", err)
	}
}

func TestLoad_CreatesDefaultConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg == nil {
		t.Fatalf("Load() returned nil config")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `workspace = "./workspace"`) {
		t.Fatalf("default config content unexpected: %s", content)
	}
	if !strings.Contains(content, "[exec]") {
		t.Fatalf("default config missing exec section: %s", content)
	}
	if !strings.Contains(content, "timeout = 60") {
		t.Fatalf("default config missing exec timeout: %s", content)
	}
}

func TestConfig_EnsureWorkspaceScaffold(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := filepath.Join(tmpDir, "workspace")
	cfg := &Config{
		Workspace: workspace,
	}

	if err := cfg.EnsureWorkspaceScaffold(); err != nil {
		t.Fatalf("EnsureWorkspaceScaffold() error = %v", err)
	}

	paths := []string{
		filepath.Join(workspace, "SOUL.md"),
		filepath.Join(workspace, "USER.md"),
		filepath.Join(workspace, "hooks", "hooks.js"),
		filepath.Join(workspace, "skills"),
		filepath.Join(workspace, "temp"),
		filepath.Join(workspace, "report"),
		filepath.Join(workspace, "reports"),
		filepath.Join(workspace, "scripts"),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected scaffold path %s: %v", path, err)
		}
	}
}

func TestConfig_EnsureWorkspaceScaffold_DoesNotOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := filepath.Join(tmpDir, "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	cfg := &Config{
		Workspace: workspace,
	}
	if err := cfg.EnsureWorkspaceScaffold(); err != nil {
		t.Fatalf("EnsureWorkspaceScaffold() error = %v", err)
	}
}

func TestConfig_GetPaths(t *testing.T) {
	tmpDir := t.TempDir()
	workspacePath := filepath.Join(tmpDir, "workspace")
	dbPath := filepath.Join(tmpDir, "data", "app.db")

	cfg := &Config{
		Workspace: workspacePath,
		Database:  DatabaseConfig{Path: dbPath},
	}

	wsAbs, err := cfg.GetWorkspacePath()
	if err != nil {
		t.Errorf("GetWorkspacePath() error = %v", err)
	}
	if wsAbs == "" {
		t.Error("GetWorkspacePath() returned empty path")
	}

	dbAbs, err := cfg.GetDatabasePath()
	if err != nil {
		t.Errorf("GetDatabasePath() error = %v", err)
	}
	if dbAbs == "" {
		t.Error("GetDatabasePath() returned empty path")
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Field:   "workspace",
		Message: "workspace 是必需的",
	}

	expected := "workspace: workspace 是必需的"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %q, want %q", err.Error(), expected)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && len(substr) > 0 && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
