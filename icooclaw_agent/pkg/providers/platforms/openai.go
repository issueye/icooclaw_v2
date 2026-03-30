package platforms

import (
	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

// OpenAIProvider implements Provider for OpenAI.
type OpenAIProvider struct {
	*openAICompatibleProvider
}

func newOpenAIProvider(
	providerName consts.ProviderType,
	cfg *storage.Provider,
	defaultAPIBase string,
	defaultModel string,
) *OpenAIProvider {
	return &OpenAIProvider{
		openAICompatibleProvider: newOpenAICompatibleProvider(providerName, cfg, openAICompatibleProfile{
			defaultAPIBase: defaultAPIBase,
			defaultModel:   defaultModel,
		}),
	}
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(cfg *storage.Provider) Provider {
	return newOpenAIProvider(
		consts.ProviderOpenAI,
		cfg,
		"https://api.openai.com/v1",
		"gpt-4o",
	)
}
