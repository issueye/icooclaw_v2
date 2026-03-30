package platforms

import (
	"testing"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

func TestNewMiniMaxAnthropicProvider(t *testing.T) {
	provider := NewMiniMaxAnthropicProvider(&storage.Provider{
		Name:     "minimax-anthropic",
		Type:     consts.ProviderMiniMax,
		Protocol: consts.ProtocolAnthropic,
		APIKey:   "test-key",
	})

	minimax, ok := provider.(*AnthropicProvider)
	if !ok {
		t.Fatalf("expected AnthropicProvider, got %T", provider)
	}
	if minimax.GetName() != consts.ProviderMiniMax.ToString() {
		t.Fatalf("provider name = %s, want %s", minimax.GetName(), consts.ProviderMiniMax)
	}
	if minimax.APIBase() != "https://api.minimax.io/anthropic" {
		t.Fatalf("apiBase = %s, want anthropic compatible endpoint", minimax.APIBase())
	}
	if minimax.GetModel() != "MiniMax-M2.5" {
		t.Fatalf("model = %s, want MiniMax-M2.5", minimax.GetModel())
	}
}

func TestNewMiniMaxOpenAIProvider(t *testing.T) {
	provider := NewMiniMaxOpenAIProvider(&storage.Provider{
		Name:     "minimax-openai",
		Type:     consts.ProviderMiniMax,
		Protocol: consts.ProtocolOpenAI,
		APIKey:   "test-key",
	})

	minimax, ok := provider.(*OpenAIProvider)
	if !ok {
		t.Fatalf("expected OpenAIProvider, got %T", provider)
	}
	if minimax.GetName() != consts.ProviderMiniMax.ToString() {
		t.Fatalf("provider name = %s, want %s", minimax.GetName(), consts.ProviderMiniMax)
	}
	if minimax.APIBase() != "https://api.minimax.io/v1" {
		t.Fatalf("apiBase = %s, want openai compatible endpoint", minimax.APIBase())
	}
}

func TestNewMiniMaxProvider_ReturnsNilForUnsupportedProtocol(t *testing.T) {
	provider := NewMiniMaxProvider(&storage.Provider{
		Name:     "minimax-invalid",
		Type:     consts.ProviderMiniMax,
		Protocol: consts.ProviderProtocol("responses"),
		APIKey:   "test-key",
	})

	if provider != nil {
		t.Fatalf("expected nil provider for unsupported protocol, got %T", provider)
	}
}
