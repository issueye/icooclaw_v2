package app

import (
	"encoding/json"

	"icooclaw/pkg/providers"
	"icooclaw/pkg/utils"
)

func mustMarshalJSON(v any) string {
	return utils.MustMarshalJSON(v)
}

func decodeHookMessages(result any) ([]providers.ChatMessage, bool, error) {
	var messages []providers.ChatMessage
	ok, err := decodeHookField(result, "messages", &messages)
	return messages, ok, err
}

func decodeHookToolCall(result any) (providers.ToolCall, bool, error) {
	var toolCall providers.ToolCall
	ok, err := decodeHookField(result, "toolCall", &toolCall)
	return toolCall, ok, err
}

func decodeHookArgs(result any) (map[string]any, bool, error) {
	var args map[string]any
	ok, err := decodeHookValue(result, &args)
	return args, ok, err
}

func decodeHookField(result any, field string, dest any) (bool, error) {
	if result == nil {
		return false, nil
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return false, nil
	}

	raw, ok := resultMap[field]
	if !ok {
		return false, nil
	}

	return decodeHookValue(raw, dest)
}

func decodeHookValue(value any, dest any) (bool, error) {
	if value == nil {
		return false, nil
	}

	switch v := value.(type) {
	case string:
		if v == "" {
			return false, nil
		}
		return true, json.Unmarshal([]byte(v), dest)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return false, err
		}
		return true, json.Unmarshal(data, dest)
	}
}
