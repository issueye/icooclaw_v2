package platforms

import (
	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

type SiliconFlowProvider struct {
	*openAICompatibleProvider
}

func NewSiliconFlowProvider(cfg *storage.Provider) Provider {
	return &SiliconFlowProvider{
		openAICompatibleProvider: newOpenAICompatibleProvider(consts.ProviderSiliconFlow, cfg, openAICompatibleProfile{
			defaultAPIBase: "https://api.siliconflow.cn/v1",
			defaultModel:   "Qwen/Qwen2.5-7B-Instruct",
		}),
	}
}
