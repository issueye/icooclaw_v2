package react

import (
	"context"
	"strings"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
)

type preparedChat struct {
	sessionKey string
	provider   providers.Provider
	modelName  string
	messages   []providers.ChatMessage
}

func sanitizeProviderMessages(messages []providers.ChatMessage) []providers.ChatMessage {
	if len(messages) == 0 {
		return nil
	}

	sanitized := make([]providers.ChatMessage, 0, len(messages))
	for _, message := range messages {
		cleaned := message
		cleaned.Role = strings.TrimSpace(cleaned.Role)
		cleaned.Content = strings.TrimSpace(cleaned.Content)
		cleaned.Name = strings.TrimSpace(cleaned.Name)
		cleaned.ToolCallID = strings.TrimSpace(cleaned.ToolCallID)
		cleaned.ToolCalls = sanitizeProviderToolCalls(cleaned.ToolCalls)

		if shouldDropProviderMessage(cleaned) {
			continue
		}

		sanitized = append(sanitized, cleaned)
	}

	return sanitized
}

func sanitizeProviderToolCalls(toolCalls []providers.ToolCall) []providers.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	sanitized := make([]providers.ToolCall, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		cleaned := toolCall
		cleaned.ID = strings.TrimSpace(cleaned.ID)
		cleaned.Type = strings.TrimSpace(cleaned.Type)
		cleaned.Function.Name = strings.TrimSpace(cleaned.Function.Name)
		cleaned.Function.Arguments = strings.TrimSpace(cleaned.Function.Arguments)

		if cleaned.Function.Name == "" {
			continue
		}
		if cleaned.Type == "" {
			cleaned.Type = "function"
		}
		if cleaned.Function.Arguments == "" {
			cleaned.Function.Arguments = "{}"
		}

		sanitized = append(sanitized, cleaned)
	}

	return sanitized
}

func shouldDropProviderMessage(message providers.ChatMessage) bool {
	switch message.Role {
	case consts.RoleAssistant.ToString():
		return message.Content == "" && len(message.ToolCalls) == 0
	case consts.RoleTool.ToString():
		return message.Content == "" || message.ToolCallID == ""
	case consts.RoleUser.ToString(), consts.RoleSystem.ToString():
		return message.Content == ""
	default:
		return message.Content == "" && len(message.ToolCalls) == 0
	}
}

func (a *ReActAgent) prepareChat(ctx context.Context, msg bus.InboundMessage) (*preparedChat, error) {
	if a.hooks != nil {
		a.hooks.OnAgentStart(ctx, msg)
	}

	sessionKey := consts.GetSessionKey(msg.Channel, msg.SessionID)

	provider, modelName, err := a.GetDynamicProvider(ctx)
	if err != nil {
		return nil, err
	}

	messages, err := a.buildMessages(ctx, sessionKey, msg)
	if err != nil {
		return nil, err
	}

	if a.memory != nil {
		if _, err := a.memory.SaveAndGetID(ctx, sessionKey, consts.RoleUser.ToString(), msg.Text); err != nil {
			a.log().With("name", "【智能体】").Warn("保存用户消息失败", "error", err)
		}
	}

	return &preparedChat{
		sessionKey: sessionKey,
		provider:   provider,
		modelName:  modelName,
		messages:   messages,
	}, nil
}
