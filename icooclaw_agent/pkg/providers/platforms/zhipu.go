package platforms

import (
	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

type ZhipuProvider struct {
	*openAICompatibleProvider
}

func NewZhipuProvider(cfg *storage.Provider) Provider {
	return &ZhipuProvider{
		openAICompatibleProvider: newOpenAICompatibleProvider(consts.ProviderZhipu, cfg, openAICompatibleProfile{
			defaultAPIBase: "https://open.bigmodel.cn/api/paas/v4",
			defaultModel:   "glm-4",
		}),
	}
}
