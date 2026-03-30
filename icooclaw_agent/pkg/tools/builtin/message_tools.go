package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
)

type messageToolBase struct {
	storage *storage.Storage
}

func newMessageToolBase(st *storage.Storage) messageToolBase {
	return messageToolBase{storage: st}
}

func (b messageToolBase) ensureStorage() error {
	if b.storage == nil || b.storage.Message() == nil {
		return fmt.Errorf("message storage is not configured")
	}
	return nil
}

func messageStringArg(args map[string]any, key string) string {
	if value, ok := args[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func parseOptionalTimeArg(args map[string]any, key string) (*time.Time, error) {
	value := messageStringArg(args, key)
	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("%s must be RFC3339 time, for example 2026-03-26T15:04:05Z", key)
	}
	return &parsed, nil
}

func marshalMessageResult(v any) *tools.Result {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("marshal result failed: %w", err)}
	}
	return &tools.Result{Success: true, Content: string(data)}
}

type SearchMessagesTool struct {
	messageToolBase
}

func NewSearchMessagesTool(st *storage.Storage) *SearchMessagesTool {
	return &SearchMessagesTool{messageToolBase: newMessageToolBase(st)}
}

func (t *SearchMessagesTool) Name() string {
	return "search_messages"
}

func (t *SearchMessagesTool) Description() string {
	return "检索历史消息，可按会话、角色、关键词和时间范围过滤，返回最近命中的消息列表。"
}

func (t *SearchMessagesTool) Parameters() map[string]any {
	return map[string]any{
		"session_id": map[string]any{
			"type":        "string",
			"description": "可选，会话 ID。",
		},
		"role": map[string]any{
			"type":        "string",
			"description": "可选，消息角色：user、assistant、system。",
			"enum":        []string{"user", "assistant", "system"},
		},
		"keyword": map[string]any{
			"type":        "string",
			"description": "可选，按 content 或 thinking 模糊搜索。",
		},
		"since": map[string]any{
			"type":        "string",
			"description": "可选，起始时间，RFC3339 格式。",
		},
		"until": map[string]any{
			"type":        "string",
			"description": "可选，结束时间，RFC3339 格式。",
		},
		"limit": map[string]any{
			"type":        "integer",
			"description": "最大返回条数，默认 20，最大 200。",
		},
		"include_summary": map[string]any{
			"type":        "boolean",
			"description": "是否包含摘要消息，默认 false。",
		},
	}
}

func (t *SearchMessagesTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	if err := t.ensureStorage(); err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	since, err := parseOptionalTimeArg(args, "since")
	if err != nil {
		return &tools.Result{Success: false, Error: err}
	}
	until, err := parseOptionalTimeArg(args, "until")
	if err != nil {
		return &tools.Result{Success: false, Error: err}
	}
	if since != nil && until != nil && since.After(*until) {
		return &tools.Result{Success: false, Error: fmt.Errorf("since must be earlier than until")}
	}

	limit := 20
	if value, ok := args["limit"]; ok {
		switch typed := value.(type) {
		case int:
			limit = typed
		case int32:
			limit = int(typed)
		case int64:
			limit = int(typed)
		case float64:
			limit = int(typed)
		}
	}
	includeSummary, _ := args["include_summary"].(bool)

	messages, err := t.storage.Message().Search(&storage.SearchMessageQuery{
		SessionID:      messageStringArg(args, "session_id"),
		Role:           messageStringArg(args, "role"),
		KeyWord:        messageStringArg(args, "keyword"),
		Since:          since,
		Until:          until,
		Limit:          limit,
		IncludeSummary: includeSummary,
	})
	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("search messages failed: %w", err)}
	}

	type messageSearchItem struct {
		ID          string    `json:"id"`
		SessionID   string    `json:"session_id"`
		Role        string    `json:"role"`
		Content     string    `json:"content"`
		Thinking    string    `json:"thinking,omitempty"`
		TotalTokens int       `json:"total_tokens,omitempty"`
		CreatedAt   time.Time `json:"created_at"`
	}

	items := make([]messageSearchItem, 0, len(messages))
	for _, msg := range messages {
		items = append(items, messageSearchItem{
			ID:          msg.ID,
			SessionID:   msg.SessionID,
			Role:        string(msg.Role),
			Content:     msg.Content,
			Thinking:    msg.Thinking,
			TotalTokens: msg.TotalTokens,
			CreatedAt:   msg.CreatedAt,
		})
	}

	return marshalMessageResult(map[string]any{
		"count":    len(items),
		"messages": items,
	})
}
