package platforms

import (
	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

// DeepSeekProvider implements Provider for DeepSeek.
type DeepSeekProvider struct {
	*openAICompatibleProvider
}

// NewDeepSeekProvider creates a new DeepSeek provider.
func NewDeepSeekProvider(cfg *storage.Provider) Provider {
	return &DeepSeekProvider{
		openAICompatibleProvider: newOpenAICompatibleProvider(consts.ProviderDeepSeek, cfg, openAICompatibleProfile{
			defaultAPIBase: "https://api.deepseek.com/v1",
			defaultModel:   "deepseek-chat",
		}),
	}
}
