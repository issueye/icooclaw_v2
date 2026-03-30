package platforms

import (
	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

type MoonshotProvider struct {
	*openAICompatibleProvider
}

func NewMoonshotProvider(cfg *storage.Provider) Provider {
	return &MoonshotProvider{
		openAICompatibleProvider: newOpenAICompatibleProvider(consts.ProviderMoonshot, cfg, openAICompatibleProfile{
			defaultAPIBase: "https://api.moonshot.cn/v1",
			defaultModel:   "moonshot-v1-8k",
		}),
	}
}
