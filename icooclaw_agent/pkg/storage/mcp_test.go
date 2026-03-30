package storage

import "testing"

func TestMCPConfig_IsSSE_BackwardCompatible(t *testing.T) {
	t.Run("new sse type", func(t *testing.T) {
		cfg := &MCPConfig{Type: MCPTypeSSE}
		if !cfg.IsSSE() {
			t.Fatalf("expected sse config to be recognized")
		}
	})

	t.Run("legacy streamable http type", func(t *testing.T) {
		cfg := &MCPConfig{Type: MCPType("Streamable HTTP")}
		if !cfg.IsSSE() {
			t.Fatalf("expected legacy Streamable HTTP config to be recognized as sse")
		}
	})
}
