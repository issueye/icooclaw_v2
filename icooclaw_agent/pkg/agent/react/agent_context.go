package react

import (
	"context"
	"fmt"
	"strings"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/storage"
)

const (
	agentIDMetadataKey        = "agent_id"
	agentListPromptHeader     = "\n\n## 当前可用 Agent 列表\n"
	currentAgentPromptHeader  = "\n\n## 当前 Agent\n"
	currentAgentPromptNameFmt = "- 名称 %s\n"
	currentAgentPromptIDFmt   = "- ID %s\n"
	currentAgentDescFmt       = "- 描述 %s\n"
	currentAgentSystemFmt     = "\n### 当前 Agent 附加系统提示词\n%s\n"
	agentListItemFmt          = "- 名称 %s | 类型 %s | ID %s\n"
)

func (a *ReActAgent) resolveAgentProfile(msg bus.InboundMessage) (*storage.Agent, error) {
	if a.storage == nil || a.storage.Agent() == nil {
		return nil, nil
	}

	agentID := strings.TrimSpace(stringMetadataValue(msg.Metadata, agentIDMetadataKey))
	if agentID != "" {
		agentInfo, err := a.storage.Agent().GetByID(agentID)
		if err == nil {
			return agentInfo, nil
		}
		a.log().Warn("按 ID 获取 agent 失败，回退默认 agent", "agent_id", agentID, "error", err)
	}

	return a.storage.ResolveDefaultAgent()
}

func (a *ReActAgent) appendAgentContext(ctx context.Context, systemPrompt string, msg bus.InboundMessage) (string, error) {
	if a.storage == nil || a.storage.Agent() == nil {
		return systemPrompt, nil
	}

	agents, err := a.storage.Agent().List()
	if err != nil {
		return "", fmt.Errorf("获取 agent 列表失败: %w", err)
	}

	sb := strings.Builder{}
	sb.WriteString(systemPrompt)
	sb.WriteString(agentListPromptHeader)
	for _, item := range agents {
		fmt.Fprintf(&sb, agentListItemFmt, item.Name, item.Type, item.ID)
	}

	currentAgent, err := a.resolveAgentProfile(msg)
	if err != nil {
		return "", fmt.Errorf("获取当前 agent 失败: %w", err)
	}
	if currentAgent != nil {
		sb.WriteString(currentAgentPromptHeader)
		fmt.Fprintf(&sb, currentAgentPromptNameFmt, currentAgent.Name)
		fmt.Fprintf(&sb, currentAgentPromptIDFmt, currentAgent.ID)
		if strings.TrimSpace(currentAgent.Description) != "" {
			fmt.Fprintf(&sb, currentAgentDescFmt, currentAgent.Description)
		}
		if strings.TrimSpace(currentAgent.SystemPrompt) != "" {
			fmt.Fprintf(&sb, currentAgentSystemFmt, currentAgent.SystemPrompt)
		}
	}

	_ = ctx
	return sb.String(), nil
}

func stringMetadataValue(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok || value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}
