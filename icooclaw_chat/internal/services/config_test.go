package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigServiceSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "icoo_chat.toml")
	service := &ConfigService{
		config:     defaultConfig(),
		configPath: path,
	}

	err := service.SetClawConnectionConfig(ClawConnectionConfig{
		APIBase: "http://127.0.0.1:116789",
		WSHost:  "127.0.0.1",
		WSPort:  "116789",
		WSPath:  "/ws",
		UserID:  "tester",
	})
	if err != nil {
		t.Fatalf("SetClawConnectionConfig() error = %v", err)
	}

	loaded := &ConfigService{
		config:     defaultConfig(),
		configPath: path,
	}
	if err := loaded.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	cfg := loaded.GetClawConnectionConfig()
	if cfg.APIBase != "http://127.0.0.1:116789" {
		t.Fatalf("APIBase = %q", cfg.APIBase)
	}
	if cfg.WSHost != "127.0.0.1" {
		t.Fatalf("WSHost = %q", cfg.WSHost)
	}
	if cfg.UserID != "tester" {
		t.Fatalf("UserID = %q", cfg.UserID)
	}
}

func TestConfigServiceLoadAppliesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "icoo_chat.toml")
	content := []byte("[claw_connection]\nws_host = \"custom-host\"\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	service := &ConfigService{
		config:     defaultConfig(),
		configPath: path,
	}
	if err := service.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	clawCfg := service.GetClawConnectionConfig()
	if clawCfg.WSHost != "custom-host" {
		t.Fatalf("WSHost = %q, want custom-host", clawCfg.WSHost)
	}
	if clawCfg.APIBase != "http://localhost:16789" {
		t.Fatalf("APIBase = %q, want default", clawCfg.APIBase)
	}
	if clawCfg.WSPort != "16789" {
		t.Fatalf("WSPort = %q, want default", clawCfg.WSPort)
	}
}

func TestConfigServiceEnsureConfigFileCreatesDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "icoo_chat.toml")
	service := &ConfigService{
		config:     defaultConfig(),
		configPath: path,
	}

	if err := service.ensureConfigFile(); err != nil {
		t.Fatalf("ensureConfigFile() error = %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
}
