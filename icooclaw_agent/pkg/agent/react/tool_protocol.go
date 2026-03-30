package react

import (
	"context"
	"encoding/json"
	"fmt"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/providers"
)

func (a *ReActAgent) executeToolCall(ctx context.Context, tc providers.ToolCall, msg bus.InboundMessage) (string, error) {
	toolName := tc.Function.Name
	var err error

	if a.hooks != nil {
		tc, err = a.hooks.OnToolCallBefore(ctx, toolName, tc, msg)
		if err != nil {
			return "", err
		}
	}

	var args map[string]any
	if tc.Function.Arguments != "" {
		if a.hooks != nil {
			args, err = a.hooks.OnToolParseArguments(ctx, toolName, tc, msg)
			if err != nil {
				return "", err
			}
		} else {
			err = json.Unmarshal([]byte(tc.Function.Arguments), &args)
			if err != nil {
				a.log().With("name", "【智能体】").Error("解析工具参数失败",
					"tool", toolName,
					"error", err)
				return "", fmt.Errorf("解析参数失败: %w", err)
			}
		}
	}

	result := a.tools.ExecuteWithContext(ctx, toolName, args, msg.Channel, msg.SessionID, nil)
	if result.Error != nil {
		return "", result.Error
	}

	if a.hooks != nil {
		err = a.hooks.OnToolCallAfter(ctx, toolName, msg, result)
		if err != nil {
			return "", err
		}
	}

	return result.Content, nil
}

func (a *ReActAgent) mergeToolCalls(toolCalls []providers.ToolCall) []providers.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	indexToCall := make(map[int]*providers.ToolCall)
	realIDToIndex := make(map[string]int)
	nextIndex := 0

	for _, tc := range toolCalls {
		var idx int
		var found bool

		if isStreamIndexID(tc.ID) {
			fmt.Sscanf(tc.ID, "stream_index:%d", &idx)
			found = true
		} else if tc.ID != "" {
			if i, ok := realIDToIndex[tc.ID]; ok {
				idx = i
				found = true
			} else {
				idx = nextIndex
				nextIndex++
				realIDToIndex[tc.ID] = idx
				found = true
			}
		}

		if !found {
			continue
		}

		if existing, ok := indexToCall[idx]; ok {
			if tc.Function.Name != "" {
				existing.Function.Name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				existing.Function.Arguments += tc.Function.Arguments
			}
			if tc.ID != "" && !isStreamIndexID(tc.ID) {
				existing.ID = tc.ID
			}
		} else {
			copy := tc
			indexToCall[idx] = &copy
		}
	}

	result := make([]providers.ToolCall, 0, len(indexToCall))
	for _, tc := range indexToCall {
		if tc.Function.Name == "" {
			continue
		}
		result = append(result, providers.ToolCall{
			ID:   tc.ID,
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}

	return result
}

func (a *ReActAgent) validateToolCalls(toolCalls []providers.ToolCall) []providers.ToolCall {
	valid := make([]providers.ToolCall, 0, len(toolCalls))

	for _, tc := range toolCalls {
		if tc.Function.Name == "" {
			a.log().Warn("跳过无效工具调用：缺少工具名称", "id", tc.ID)
			continue
		}

		if tc.Function.Arguments == "" {
			tc.Function.Arguments = "{}"
		}

		valid = append(valid, tc)
	}

	return valid
}

func isStreamIndexID(id string) bool {
	return len(id) > 12 && id[:12] == "stream_index"
}
