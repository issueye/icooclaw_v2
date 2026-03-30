package react

import (
	"context"
	"fmt"
	"strings"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/tools"
)

// buildMessages 构建 LLM 请求的消息列表。
func (a *ReActAgent) buildMessages(ctx context.Context, sessionKey string, msg bus.InboundMessage) ([]providers.ChatMessage, error) {
	var (
		messages = make([]providers.ChatMessage, 0)
		err      error
	)

	if a.hooks != nil {
		messages, err = a.hooks.OnBuildMessagesBefore(ctx, sessionKey, msg, messages)
		if err != nil {
			return nil, err
		}
	}

	systemPrompt, err := a.storage.Workspace().LoadWorkspace()
	if err != nil {
		return nil, err
	}
	systemPrompt, err = a.appendAgentContext(ctx, systemPrompt, msg)
	if err != nil {
		return nil, err
	}

	skills, err := a.skills.List(ctx)
	if err != nil {
		return nil, err
	}

	sb := strings.Builder{}
	sb.WriteString("\n\n## 技能列表\n")
	for _, skill := range skills {
		displayName := skill.Title
		if displayName == "" {
			displayName = skill.Name
		}
		fmt.Fprintf(&sb, "- 名称 %s [%s]\n", displayName, skill.Name)
		fmt.Fprintf(&sb, "		描述 %s\n", skill.Description)
	}

	systemPrompt += sb.String()
	messages = append(messages, providers.ChatMessage{
		Role:    consts.RoleSystem.ToString(),
		Content: systemPrompt,
	})

	var history []providers.ChatMessage
	if a.memory != nil {
		mem, err := a.memory.Load(ctx, sessionKey)
		if err != nil {
			a.log().With("name", "【智能体】").Warn("加载记忆失败", "error", err, "session_key", sessionKey)
		} else {
			history = mem
		}
	}
	messages = append(messages, history...)

	messages = append(messages, providers.ChatMessage{
		Role:    consts.RoleUser.ToString(),
		Content: msg.Text,
	})

	if a.hooks != nil {
		messages, err = a.hooks.OnBuildMessagesAfter(ctx, sessionKey, msg, messages)
		if err != nil {
			return nil, err
		}
	}

	return messages, nil
}

// convertToolDefinitions 将工具定义转换为提供商工具格式。
func (a *ReActAgent) convertToolDefinitions(defs []tools.ToolDefinition) []providers.Tool {
	tools := make([]providers.Tool, 0, len(defs))
	for _, def := range defs {
		tools = append(tools, providers.Tool{
			Type: def.Type,
			Function: providers.Function{
				Name:        def.Function.Name,
				Description: def.Function.Description,
				Parameters:  def.Function.Parameters,
			},
		})
	}
	return tools
}
