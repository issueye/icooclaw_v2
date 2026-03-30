package handlers

import (
	"log/slog"
	"testing"

	"icooclaw/pkg/storage"
)

func TestNormalizeMCPConfig(t *testing.T) {
	t.Run("stdio config trims and clears sse fields", func(t *testing.T) {
		cfg := &storage.MCPConfig{
			Name:           "  demo-stdio  ",
			Description:    "  test server  ",
			Type:           storage.MCPType(" STDIO "),
			Command:        "  npx  ",
			URL:            "  https://example.com/sse  ",
			RetryCount:     0,
			TimeoutSeconds: 0,
		}

		if err := normalizeMCPConfig(cfg); err != nil {
			t.Fatalf("normalizeMCPConfig() error = %v", err)
		}

		if cfg.Name != "demo-stdio" {
			t.Fatalf("cfg.Name = %q, want %q", cfg.Name, "demo-stdio")
		}
		if cfg.Type != storage.MCPTypeStdio {
			t.Fatalf("cfg.Type = %q, want %q", cfg.Type, storage.MCPTypeStdio)
		}
		if cfg.URL != "" {
			t.Fatalf("cfg.URL = %q, want empty", cfg.URL)
		}
		if cfg.RetryCount != 3 {
			t.Fatalf("cfg.RetryCount = %d, want 3", cfg.RetryCount)
		}
		if cfg.TimeoutSeconds != 30 {
			t.Fatalf("cfg.TimeoutSeconds = %d, want 30", cfg.TimeoutSeconds)
		}
	})

	t.Run("legacy streamable http maps to sse", func(t *testing.T) {
		cfg := &storage.MCPConfig{
			Name:    "demo-sse",
			Type:    storage.MCPType("Streamable HTTP"),
			URL:     " https://example.com/mcp ",
			Args:    storage.StringArray{"--inspect"},
			Env:     map[string]string{"TOKEN": "secret"},
			Headers: map[string]string{"Authorization": "Bearer token"},
		}

		if err := normalizeMCPConfig(cfg); err != nil {
			t.Fatalf("normalizeMCPConfig() error = %v", err)
		}

		if cfg.Type != storage.MCPTypeSSE {
			t.Fatalf("cfg.Type = %q, want %q", cfg.Type, storage.MCPTypeSSE)
		}
		if cfg.Command != "" {
			t.Fatalf("cfg.Command = %q, want empty", cfg.Command)
		}
		if len(cfg.Args) != 0 {
			t.Fatalf("cfg.Args len = %d, want 0", len(cfg.Args))
		}
		if len(cfg.Env) != 0 {
			t.Fatalf("cfg.Env len = %d, want 0", len(cfg.Env))
		}
		if got := cfg.Headers["Authorization"]; got != "Bearer token" {
			t.Fatalf("cfg.Headers[Authorization] = %q, want %q", got, "Bearer token")
		}
	})

	t.Run("invalid type returns error", func(t *testing.T) {
		cfg := &storage.MCPConfig{
			Name: "invalid",
			Type: storage.MCPType("socket"),
		}

		if err := normalizeMCPConfig(cfg); err == nil {
			t.Fatalf("normalizeMCPConfig() error = nil, want error")
		}
	})
}

func TestMCPHandlerRuntimeInfoWithoutManager(t *testing.T) {
	handler := NewMCPHandler(slog.Default(), nil, nil)
	cfg := &storage.MCPConfig{
		Model:   storage.Model{ID: "mcp-1"},
		Name:    "demo",
		Enabled: true,
	}

	info := handler.runtimeInfoFor(cfg)
	if info.ID != "mcp-1" {
		t.Fatalf("info.ID = %q, want %q", info.ID, "mcp-1")
	}
	if info.State != "disconnected" {
		t.Fatalf("info.State = %q, want %q", info.State, "disconnected")
	}
	if info.Managed {
		t.Fatalf("info.Managed = true, want false")
	}
	if !info.Enabled {
		t.Fatalf("info.Enabled = false, want true")
	}
}
