package react

import (
	"context"
	"fmt"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
)

const (
	summaryConversationLineFormat = "%s: %s\n\n"
	summaryPromptTemplate         = `请对以下对话进行简洁的摘要，提取关键信息和要点。

对话内容：
%s

请用中文生成一个简洁的摘要，概括本次对话的主题、用户的需求和AI的回复要点。摘要应该清晰、准确，长度适中（50-200字）。`
	errSummaryLLMRequestFailed = "LLM请求失败: %w"
)

func (a *ReActAgent) GenerateSummary(ctx context.Context, messages []providers.ChatMessage) (string, error) {
	// 构建摘要请求
	var conversation string
	for _, m := range messages {
		conversation += fmt.Sprintf(summaryConversationLineFormat, m.Role, m.Content)
	}

	summaryPrompt := fmt.Sprintf(summaryPromptTemplate, conversation)

	// 调用 LLM 生成摘要
	provider, modelName, err := a.GetDynamicProvider(ctx)
	if err != nil {
		return "", err
	}

	// 1. 构建请求消息
	req := providers.ChatRequest{
		Model: modelName,
		Messages: []providers.ChatMessage{
			{
				Role:    consts.RoleUser.ToString(),
				Content: summaryPrompt,
			},
		},
	}

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf(errSummaryLLMRequestFailed, err)
	}

	return resp.Content, nil
}
