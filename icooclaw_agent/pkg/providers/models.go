package providers

import "icooclaw/pkg/providers/platforms"

type ModelInfo = platforms.ModelInfo

func GetModelInfo(modelID string) *ModelInfo {
	return platforms.GetModelInfo(modelID)
}

func ListModels() []*ModelInfo {
	return platforms.ListModels()
}

func ListModelsByProvider(provider string) []*ModelInfo {
	return platforms.ListModelsByProvider(provider)
}

func CalculateCost(modelID string, inputTokens, outputTokens int) (float64, error) {
	return platforms.CalculateCost(modelID, inputTokens, outputTokens)
}
