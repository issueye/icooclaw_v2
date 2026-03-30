package platforms

import (
	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

type MistralProvider struct {
	*openAICompatibleProvider
}

func NewMistralProvider(cfg *storage.Provider) Provider {
	return &MistralProvider{
		openAICompatibleProvider: newOpenAICompatibleProvider(consts.ProviderMistral, cfg, openAICompatibleProfile{
			defaultAPIBase: "https://api.mistral.ai/v1",
			defaultModel:   "mistral-large-latest",
		}),
	}
}
