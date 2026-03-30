// Package memory 提供 icooclaw 的记忆管理能力。
package memory

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"icooclaw/pkg/utils"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/storage"
)

// Loader 定义会话记忆的加载与保存接口。
type Loader interface {
	Load(ctx context.Context, sessionKey string) ([]providers.ChatMessage, error)
	Save(ctx context.Context, sessionKey, role, content string) error
	SaveAndGetID(ctx context.Context, sessionKey, role, content string) (string, error)
	Clear(ctx context.Context, sessionKey string) error
}

// DefaultLoader 是默认的记忆加载器实现。
type DefaultLoader struct {
	storage     *storage.Storage
	maxItems    int
	recentCount int // 最近消息数量
	logger      *slog.Logger
}

// NewLoader 创建一个新的记忆加载器。
func NewLoader(s *storage.Storage, maxItems int, logger *slog.Logger) *DefaultLoader {
	if maxItems <= 0 {
		maxItems = 100
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &DefaultLoader{
		storage:     s,
		maxItems:    maxItems,
		recentCount: 5, // 默认最近5条消息
		logger:      logger,
	}
}

// NewLoaderWithRecentCount 创建一个可自定义最近消息数量的记忆加载器。
func NewLoaderWithRecentCount(s *storage.Storage, maxItems, recentCount int, logger *slog.Logger) *DefaultLoader {
	if maxItems <= 0 {
		maxItems = 100
	}
	if recentCount <= 0 {
		recentCount = 5
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &DefaultLoader{
		storage:     s,
		maxItems:    maxItems,
		recentCount: recentCount,
		logger:      logger,
	}
}

// Load 加载指定会话的记忆。
// 返回值由最近消息和历史摘要组成，以减少 token 消耗。
func (l *DefaultLoader) Load(ctx context.Context, sessionKey string) ([]providers.ChatMessage, error) {
	var messages []providers.ChatMessage
	_ = ctx

	// 1. 获取最近几段历史摘要（如果有），合并成一条系统消息，减少 token 消耗。
	summaryLimit := 3
	if l.maxItems > l.recentCount && l.maxItems-l.recentCount < summaryLimit {
		summaryLimit = l.maxItems - l.recentCount
	}
	if summaryLimit <= 0 {
		summaryLimit = 1
	}

	summaryMemories, err := l.storage.Message().GetRecentSummary(sessionKey, summaryLimit)
	if err != nil {
		l.logger.Warn("获取历史摘要失败", "error", err, "session_key", sessionKey)
	}

	if summaryMessage := joinSummaryMessages(summaryMemories); summaryMessage != "" {
		messages = append(messages, providers.ChatMessage{
			Role:    consts.RoleSystem.ToString(),
			Content: summaryMessage,
		})
	}

	// 2. 获取最近 N 条消息
	recentMemories, err := l.storage.Message().Get(sessionKey, l.recentCount)
	if err != nil {
		return nil, err
	}

	// 按时间正序排列（摘要 -> 最近消息）
	for i := len(recentMemories) - 1; i >= 0; i-- {
		m := recentMemories[i]
		messages = append(messages, providers.ChatMessage{
			Role:    m.Role.ToString(),
			Content: m.Content,
		})
	}

	return messages, nil
}

// Save 保存一条记忆记录。
func (l *DefaultLoader) Save(ctx context.Context, sessionKey, role, content string) error {
	return l.storage.Message().Save(&storage.Message{
		SessionID: sessionKey,
		Role:      consts.ToRole(role),
		Content:   content,
	})
}

// SaveAndGetID 保存一条记忆记录并返回消息 ID。
func (l *DefaultLoader) SaveAndGetID(ctx context.Context, sessionKey, role, content string) (string, error) {
	msg := &storage.Message{
		SessionID: sessionKey,
		Role:      consts.ToRole(role),
		Content:   content,
	}
	if err := l.storage.Message().Save(msg); err != nil {
		return "", err
	}
	return msg.ID, nil
}

// Clear 清空指定会话的记忆。
func (l *DefaultLoader) Clear(ctx context.Context, sessionKey string) error {
	return l.storage.Message().Delete(sessionKey)
}

// Summarizer 定义会话摘要生成接口。
type Summarizer interface {
	Summarize(ctx context.Context, messages []providers.ChatMessage) (string, error)
}

// DefaultSummarizer 使用 LLM 生成摘要。
type DefaultSummarizer struct {
	provider providers.Provider
	model    string
	logger   *slog.Logger
}

// NewSummarizer 创建一个新的摘要器。
func NewSummarizer(p providers.Provider, model string, logger *slog.Logger) *DefaultSummarizer {
	if logger == nil {
		logger = slog.Default()
	}
	return &DefaultSummarizer{
		provider: p,
		model:    model,
		logger:   logger,
	}
}

// Summarize 生成对话摘要。
func (s *DefaultSummarizer) Summarize(ctx context.Context, messages []providers.ChatMessage) (string, error) {
	// 构建摘要提示词
	var content string
	for _, m := range messages {
		content += m.Role + ": " + m.Content + "\n"
	}

	req := providers.ChatRequest{
		Model: s.model,
		Messages: []providers.ChatMessage{
			{
				Role: consts.RoleSystem.ToString(),
				Content: "You are a helpful assistant that summarizes conversations. " +
					"Provide a concise summary of the key points discussed.",
			},
			{
				Role:    consts.RoleUser.ToString(),
				Content: "Please summarize this conversation:\n\n" + content,
			},
		},
	}

	resp, err := s.provider.Chat(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// Manager 管理带摘要能力的记忆系统。
type Manager struct {
	loader     Loader
	summarizer Summarizer
	storage    *storage.Storage
	maxItems   int
	logger     *slog.Logger
}

// NewManager 创建一个新的记忆管理器。
func NewManager(loader Loader, summarizer Summarizer, s *storage.Storage, maxItems int, logger *slog.Logger) *Manager {
	if maxItems <= 0 {
		maxItems = 100
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{
		loader:     loader,
		summarizer: summarizer,
		storage:    s,
		maxItems:   maxItems,
		logger:     logger,
	}
}

// Load 加载指定会话的记忆。
func (m *Manager) Load(ctx context.Context, sessionKey string) ([]providers.ChatMessage, error) {
	return m.loader.Load(ctx, sessionKey)
}

// Save 保存一条记忆记录。
func (m *Manager) Save(ctx context.Context, sessionKey, role, content string) error {
	return m.loader.Save(ctx, sessionKey, role, content)
}

// Clear 清空指定会话的记忆。
func (m *Manager) Clear(ctx context.Context, sessionKey string) error {
	return m.loader.Clear(ctx, sessionKey)
}

func joinSummaryMessages(messages []*storage.Message) string {
	if len(messages) == 0 {
		return ""
	}

	var parts []string
	for _, m := range messages {
		content := strings.TrimSpace(m.Content)
		if content != "" {
			parts = append(parts, content)
		}
	}
	if len(parts) == 0 {
		return ""
	}

	if len(parts) == 1 {
		return "历史摘要：\n" + parts[0]
	}

	var sb strings.Builder
	sb.WriteString("历史摘要：\n")
	for i, part := range parts {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, part))
	}
	return strings.TrimSpace(sb.String())
}

func mustMarshalJSON(v any) string {
	return utils.MustMarshalJSON(v)
}
