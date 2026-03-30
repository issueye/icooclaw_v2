package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/scheduler"
	"icooclaw/pkg/storage"

	"github.com/google/uuid"
)

type summaryMetadata struct {
	Type           string `json:"type"`
	SourceCount    int    `json:"source_count"`
	UntilMessageID string `json:"until_message_id"`
	UntilCreatedAt string `json:"until_created_at"`
}

type summaryTaskPayload struct {
	SessionKey     string                  `json:"session_key"`
	SourceCount    int                     `json:"source_count"`
	UntilMessageID string                  `json:"until_message_id"`
	UntilCreatedAt string                  `json:"until_created_at"`
	Messages       []providers.ChatMessage `json:"messages"`
	HookMessage    bus.InboundMessage      `json:"hook_message"`
	Response       string                  `json:"response"`
	Iteration      int                     `json:"iteration"`
}

func (h *AppAgentHooks) maybeSummarizeSession(ctx context.Context, msg bus.InboundMessage, sessionKey, response string, iteration int) (string, error) {
	lastSummaryAt, err := h.lastSummaryCutoff(sessionKey)
	if err != nil {
		return "", err
	}

	messages, err := h.storage.Message().ListSince(sessionKey, lastSummaryAt)
	if err != nil {
		return "", err
	}
	if len(messages) <= h.keepRecent {
		return "", nil
	}

	summarizeCount := len(messages) - h.keepRecent
	if summarizeCount > h.summaryBatch {
		summarizeCount = h.summaryBatch
	}
	if summarizeCount <= 0 {
		return "", nil
	}

	chunk := messages[:summarizeCount]
	chatMessages := make([]providers.ChatMessage, 0, len(chunk))
	for _, message := range chunk {
		chatMessages = append(chatMessages, providers.ChatMessage{
			Role:    message.Role.ToString(),
			Content: message.Content,
		})
	}

	lastMessage := chunk[len(chunk)-1]
	payload, err := json.Marshal(summaryTaskPayload{
		SessionKey:     sessionKey,
		SourceCount:    len(chunk),
		UntilMessageID: lastMessage.ID,
		UntilCreatedAt: lastMessage.CreatedAt.Format(time.RFC3339Nano),
		Messages:       chatMessages,
		HookMessage:    msg,
		Response:       response,
		Iteration:      iteration,
	})
	if err != nil {
		return "", err
	}

	task := &storage.Task{
		Name:        fmt.Sprintf("summary-%s", uuid.NewString()),
		Description: fmt.Sprintf("会话摘要任务(%s)", sessionKey),
		TaskType:    scheduler.TaskTypeImmediate,
		Executor:    scheduler.TaskExecutorSummary,
		Params:      string(payload),
		Enabled:     true,
		LastStatus:  scheduler.TaskStatusPending,
	}
	if err := h.storage.Task().Create(task); err != nil {
		return "", err
	}
	if h.scheduler == nil {
		return "", fmt.Errorf("scheduler is nil")
	}
	if _, err := h.scheduler.ApplyStorageTask(task); err != nil {
		return "", err
	}

	h.logger.Info("会话摘要任务已投递", "session_key", sessionKey, "task_id", task.ID, "source_count", len(chunk))
	return "", nil
}

func (h *AppAgentHooks) ExecuteSummaryTask(ctx context.Context, task *scheduler.Task) (string, error) {
	if h.summaryAgent == nil {
		return "", fmt.Errorf("summary agent is nil")
	}

	var payload summaryTaskPayload
	if err := json.Unmarshal([]byte(task.Params), &payload); err != nil {
		return "", fmt.Errorf("解析摘要任务参数失败: %w", err)
	}

	summary, err := h.summaryAgent.GenerateSummary(ctx, payload.Messages)
	if err != nil {
		return "", err
	}

	metadata := mustMarshalJSON(summaryMetadata{
		Type:           consts.SummaryMetadataType,
		SourceCount:    payload.SourceCount,
		UntilMessageID: payload.UntilMessageID,
		UntilCreatedAt: payload.UntilCreatedAt,
	})

	if err := h.storage.Message().Save(&storage.Message{
		SessionID: payload.SessionKey,
		Role:      consts.RoleSystem,
		Content:   summary,
		Summary:   summary,
		Metadata:  metadata,
	}); err != nil {
		return "", err
	}

	msgJSON := mustMarshalJSON(payload.HookMessage)
	if _, err := h.executeHook(ctx, consts.HookScriptDir, consts.HookAgentEndWithSummary, msgJSON, payload.Response, payload.Iteration, summary); err != nil {
		h.logger.Warn("执行 onAgentEndWithSummary 钩子失败", "error", err)
	}

	h.logger.Info("会话摘要已更新", "session_key", payload.SessionKey, "source_count", payload.SourceCount, "task_id", task.ID)
	return summary, nil
}

func (h *AppAgentHooks) lastSummaryCutoff(sessionKey string) (*time.Time, error) {
	summaries, err := h.storage.Message().GetSummary(sessionKey)
	if err != nil {
		return nil, err
	}
	if len(summaries) == 0 {
		return nil, nil
	}

	last := summaries[len(summaries)-1]
	if last.Metadata == "" {
		return nil, nil
	}

	var metadata summaryMetadata
	if err := json.Unmarshal([]byte(last.Metadata), &metadata); err != nil {
		return nil, nil
	}
	if metadata.UntilCreatedAt == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, metadata.UntilCreatedAt)
	if err != nil {
		return nil, nil
	}
	return &parsed, nil
}
