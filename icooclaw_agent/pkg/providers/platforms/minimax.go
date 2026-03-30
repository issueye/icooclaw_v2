package platforms

import (
	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

// NewMiniMaxProvider creates a MiniMax provider using the explicit protocol configured on the provider.
func NewMiniMaxProvider(cfg *storage.Provider) Provider {
	if cfg == nil {
		return nil
	}

	switch cfg.Protocol {
	case consts.ProtocolOpenAI:
		return NewMiniMaxOpenAIProvider(cfg)
	case consts.ProtocolAnthropic:
		return NewMiniMaxAnthropicProvider(cfg)
	default:
		return nil
	}
}

// NewMiniMaxOpenAIProvider creates a MiniMax provider using the OpenAI-compatible API.
func NewMiniMaxOpenAIProvider(cfg *storage.Provider) Provider {
	return newOpenAIProvider(
		consts.ProviderMiniMax,
		cfg,
		"https://api.minimax.io/v1",
		"MiniMax-M2.5",
	)
}

// NewMiniMaxAnthropicProvider creates a MiniMax provider using the Anthropic-compatible API.
func NewMiniMaxAnthropicProvider(cfg *storage.Provider) Provider {
	return newAnthropicProvider(
		consts.ProviderMiniMax,
		cfg,
		"https://api.minimax.io/anthropic",
		"MiniMax-M2.5",
	)
}
