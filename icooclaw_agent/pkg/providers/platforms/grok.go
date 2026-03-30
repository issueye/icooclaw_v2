package platforms

import (
	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

type GrokProvider struct {
	*openAICompatibleProvider
}

func NewGrokProvider(cfg *storage.Provider) Provider {
	return &GrokProvider{
		openAICompatibleProvider: newOpenAICompatibleProvider(consts.ProviderGrok, cfg, openAICompatibleProfile{
			defaultAPIBase: "https://api.x.ai/v1",
			defaultModel:   "grok-2-latest",
		}),
	}
}
