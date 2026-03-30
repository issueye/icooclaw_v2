package utils

import (
	"fmt"
	"regexp"
	"strings"
)

var skillIdentifierSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

// SplitProviderModel 分割模型字符串，格式为 "provider/model"。
func SplitProviderModel(modelStr string) []string {
	idx := -1
	for i := 0; i < len(modelStr); i++ {
		if modelStr[i] == '/' {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil
	}
	return []string{modelStr[:idx], modelStr[idx+1:]}
}

// ValidateSkillIdentifier validates that the given skill identifier (slug or registry name) is non-empty
// and does not contain path separators ("/", "\\") or ".." for security.
func ValidateSkillIdentifier(identifier string) error {
	trimmed := strings.TrimSpace(identifier)
	if trimmed == "" {
		return fmt.Errorf("identifier is required and must be a non-empty string")
	}
	if strings.ContainsAny(trimmed, "/\\") || strings.Contains(trimmed, "..") {
		return fmt.Errorf("identifier must not contain path separators or '..' to prevent directory traversal")
	}
	return nil
}

// NormalizeSkillIdentifier converts display names such as "AMap Weather" or "baidu_search"
// into a stable slug form like "amap-weather" or "baidu-search".
func NormalizeSkillIdentifier(identifier string) string {
	trimmed := strings.TrimSpace(strings.ToLower(identifier))
	if trimmed == "" {
		return ""
	}

	normalized := skillIdentifierSanitizer.ReplaceAllString(trimmed, "-")
	return strings.Trim(normalized, "-")
}
