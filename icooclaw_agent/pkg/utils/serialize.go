package utils

import "encoding/json"

// parseConfig 解析 JSON 配置字符串
func ParseConfig(configStr string) map[string]any {
	result := make(map[string]any)
	if configStr == "" {
		return result
	}

	// 解析 JSON 配置字符串
	if err := json.Unmarshal([]byte(configStr), &result); err != nil {
		// 记录错误但返回空 map
		return make(map[string]any)
	}
	return result
}
