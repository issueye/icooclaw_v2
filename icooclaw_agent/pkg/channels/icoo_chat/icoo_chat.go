package icoo_chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels/errs"
	"icooclaw/pkg/channels/models"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/utils"

	"github.com/gorilla/websocket"
)

type Channel struct {
	config Config
	bus    *bus.MessageBus
	logger *slog.Logger

	conn   *websocket.Conn
	cancel context.CancelFunc

	writeMu sync.Mutex
	mu      sync.RWMutex
	running atomic.Bool
}

type wsEvent struct {
	Type      string         `json:"type"`
	SessionID string         `json:"session_id,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	Error     map[string]any `json:"error,omitempty"`
}

func New(logger *slog.Logger, msgBus *bus.MessageBus, cfgStr string) (models.Channel, error) {
	if logger == nil {
		logger = slog.Default()
	}

	cfgMap := utils.ParseConfig(cfgStr)
	cfg, err := ParseConfig(cfgMap)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &Channel{
		config: cfg,
		bus:    msgBus,
		logger: logger,
	}, nil
}

func (c *Channel) Start(ctx context.Context) error {
	if c.bus == nil {
		return fmt.Errorf("icoo_chat bus is nil")
	}

	conn, err := c.connect(ctx)
	if err != nil {
		return err
	}

	runCtx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.conn = conn
	c.cancel = cancel
	c.mu.Unlock()
	c.running.Store(true)
	c.logger.With("name", "【icoo_chat】").Debug("启动 icoo_chat bot websocket 读取循环")
	go c.readLoop(runCtx)
	return nil
}

func (c *Channel) Stop(ctx context.Context) error {
	_ = ctx
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()
	c.running.Store(false)
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (c *Channel) Send(ctx context.Context, msg models.OutboundMessage) error {
	if !c.IsRunning() {
		return errs.ErrNotRunning
	}
	if strings.TrimSpace(msg.SessionID) == "" {
		return fmt.Errorf("session ID is required")
	}

	if payloads := buildStreamPayloads(msg); len(payloads) > 0 {
		for _, payload := range payloads {
			if err := c.writeJSON(ctx, payload); err != nil {
				return err
			}
		}
		return nil
	}

	payload := map[string]any{
		"type": "reply",
		"data": map[string]any{
			"session_id": msg.SessionID,
			"content":    msg.Text,
		},
	}
	return c.writeJSON(ctx, payload)
}

func (c *Channel) IsRunning() bool {
	return c.running.Load()
}

func (c *Channel) IsAllowed(senderID string) bool {
	if len(c.config.AllowFrom) == 0 {
		return true
	}
	for _, allowed := range c.config.AllowFrom {
		if senderID == allowed {
			return true
		}
	}
	return false
}

func (c *Channel) connect(ctx context.Context) (*websocket.Conn, error) {
	u, err := url.Parse(c.config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid icoo_chat endpoint: %w", err)
	}

	query := u.Query()
	query.Set("app_id", c.config.AppID)
	query.Set("app_secret", c.config.AppSecret)
	u.RawQuery = query.Encode()

	c.logger.With("name", "【icoo_chat】").Debug("连接 icoo_chat bot websocket", "url", u.String())

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), http.Header{})
	if err != nil {
		return nil, fmt.Errorf("connect icoo_chat bot websocket failed: %w", err)
	}
	c.logger.With("name", "【icoo_chat】").Debug("连接 icoo_chat bot websocket 成功", "url", u.String())
	return conn, nil
}

func (c *Channel) readLoop(ctx context.Context) {
	defer c.running.Store(false)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn := c.getConn()
		if conn == nil {
			return
		}

		_, payload, err := conn.ReadMessage()
		if err != nil {
			c.logger.With("name", "【icoo_chat】").Warn("读取 bot websocket 失败", "error", err)
			return
		}

		var event wsEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			c.logger.With("name", "【icoo_chat】").Warn("解析 bot websocket 消息失败", "error", err)
			continue
		}
		c.handleEvent(ctx, event)
	}
}

func (c *Channel) handleEvent(ctx context.Context, event wsEvent) {
	c.logger.With("name", "【icoo_chat】").Debug("收到 bot websocket 消息", "event", event)

	switch event.Type {
	case "connected", "pong", "chunk", "end", "error":
		return
	case "message":
		data := event.Data
		if data == nil {
			return
		}
		senderID := firstString(data["from_user_id"], data["from_device_id"])
		if !c.IsAllowed(senderID) {
			return
		}

		sessionID := firstString(data["session_id"], event.SessionID)
		if sessionID == "" {
			return
		}

		metadata := mapValue(data["metadata"])
		if metadata == nil {
			metadata = map[string]any{}
		}
		metadata["conversation_type"] = firstString(data["conversation_type"])
		metadata["bot_id"] = firstString(data["bot_id"])
		metadata["bot_name"] = firstString(data["bot_name"])
		metadata["app_id"] = firstString(data["app_id"])
		if messageID := firstString(data["message_id"]); messageID != "" {
			metadata["message_id"] = messageID
		}

		inbound := bus.InboundMessage{
			Channel:   consts.ICOO_CHAT,
			SessionID: sessionID,
			Sender: bus.SenderInfo{
				ID:   senderID,
				Name: senderID,
			},
			Text:      firstString(data["content"]),
			Timestamp: time.Now(),
			Metadata:  metadata,
		}

		pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := c.bus.PublishInbound(pubCtx, inbound); err != nil {
			c.logger.With("name", "【icoo_chat】").Error("发布入站消息失败", "error", err)
		}
	}
}

func (c *Channel) writeJSON(ctx context.Context, payload any) error {
	conn := c.getConn()
	if conn == nil {
		return errs.ErrNotRunning
	}

	done := make(chan error, 1)
	go func() {
		c.writeMu.Lock()
		defer c.writeMu.Unlock()
		done <- conn.WriteJSON(payload)
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("write icoo_chat websocket failed: %w", err)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Channel) getConn() *websocket.Conn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

func firstString(values ...any) string {
	for _, value := range values {
		if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func mapValue(value any) map[string]any {
	if value == nil {
		return nil
	}
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return nil
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	if value, ok := metadata[key].(string); ok {
		return value
	}
	return ""
}

func buildStreamPayloads(msg models.OutboundMessage) []map[string]any {
	eventType := metadataString(msg.Metadata, "event_type")
	switch eventType {
	case "chunk":
		data := map[string]any{
			"session_id": msg.SessionID,
			"content":    msg.Text,
		}
		if reasoning := metadataString(msg.Metadata, "reasoning"); reasoning != "" {
			data["reasoning"] = reasoning
		}
		if iteration, ok := msg.Metadata["iteration"]; ok {
			data["iteration"] = iteration
		}
		return []map[string]any{{
			"type":       "chunk",
			"session_id": msg.SessionID,
			"data":       data,
		}}
	case "end":
		return []map[string]any{{
			"type":       "end",
			"session_id": msg.SessionID,
		}}
	case "error":
		return []map[string]any{
			{
				"type":       "chunk",
				"session_id": msg.SessionID,
				"data": map[string]any{
					"session_id": msg.SessionID,
					"content":    firstString(msg.Text, metadataString(msg.Metadata, "message")),
				},
			},
			{
				"type":       "end",
				"session_id": msg.SessionID,
			},
		}
	default:
		return nil
	}
}
