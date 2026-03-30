package models

import (
	"strings"
)

type Allow []string

func IsSenderAllowed(allowFrom Allow, senderID string) bool {
	if len(allowFrom) == 0 {
		return true
	}
	for _, allowed := range allowFrom {
		if senderID == allowed {
			return true
		}
	}
	return false
}

func ParseAllowFrom(raw any) Allow {
	if raw == nil {
		return nil
	}
	if arr, ok := raw.([]any); ok {
		var result Allow
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

func ParseStringField(raw map[string]any, key string) string {
	if v, ok := raw[key]; ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func ParseBoolField(raw map[string]any, key string) bool {
	if v, ok := raw[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
