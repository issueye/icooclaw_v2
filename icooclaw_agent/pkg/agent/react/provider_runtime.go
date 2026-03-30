package react

import (
	"context"
	"fmt"
	"log/slog"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/utils"
)

// GetDynamicProvider 从存储配置动态获取提供商。
// 返回提供商实例、模型名称和错误。
func (a *ReActAgent) GetDynamicProvider(ctx context.Context) (providers.Provider, string, error) {
	if a.providerManager == nil || a.storage == nil {
		return nil, "", fmt.Errorf("未配置提供商管理器或存储")
	}

	defaultModel, err := a.storage.Param().Get(consts.DEFAULT_MODEL_KEY)
	if err != nil || defaultModel == nil || defaultModel.Value == "" {
		return nil, "", fmt.Errorf("默认模型未配置")
	}

	parts := utils.SplitProviderModel(defaultModel.Value)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("默认模型格式错误: %s", defaultModel.Value)
	}

	a.log().Info("默认模型配置", slog.String("model", defaultModel.Value))

	providerName, modelName := parts[0], parts[1]
	a.log().Info("提供商名称和模型名称", slog.String("provider", providerName), slog.String("model", modelName))

	provider, err := a.providerManager.Get(providerName)
	if err != nil {
		return nil, "", fmt.Errorf("获取Provider失败: %w", err)
	}

	if a.hooks != nil {
		err = a.hooks.OnGetProvider(ctx, providerName, a.storage.Provider())
		if err != nil {
			return nil, "", err
		}
	}

	return provider, modelName, nil
}

func (a *ReActAgent) log() *slog.Logger {
	if a.logger != nil {
		return a.logger
	}
	return slog.Default()
}
