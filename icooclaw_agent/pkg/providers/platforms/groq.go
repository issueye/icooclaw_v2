package platforms

import (
	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

type GroqProvider struct {
	*openAICompatibleProvider
}

func NewGroqProvider(cfg *storage.Provider) Provider {
	return &GroqProvider{
		openAICompatibleProvider: newOpenAICompatibleProvider(consts.ProviderGroq, cfg, openAICompatibleProfile{
			defaultAPIBase: "https://api.groq.com/openai/v1",
			defaultModel:   "llama-3.3-70b-versatile",
		}),
	}
}
