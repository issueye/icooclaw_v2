package platforms

import (
	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

// QwenProvider implements Provider for Alibaba Qwen (通义千问).
type QwenProvider struct {
	*openAICompatibleProvider
}

// NewQwenProvider creates a new Qwen provider.
func NewQwenProvider(cfg *storage.Provider) Provider {
	providerName := consts.ProviderQwen
	apiBase := "https://dashscope.aliyuncs.com/compatible-mode/v1"

	if cfg.Type == consts.ProviderQwenCodingPlan {
		providerName = consts.ProviderQwenCodingPlan
		apiBase = "https://coding.dashscope.aliyuncs.com/v1"
	}

	return &QwenProvider{
		openAICompatibleProvider: newOpenAICompatibleProvider(providerName, cfg, openAICompatibleProfile{
			defaultAPIBase: apiBase,
			defaultModel:   "qwen-plus",
		}),
	}
}
