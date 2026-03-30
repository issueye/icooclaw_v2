// Package websocket 提供网关侧 WebSocket 连接管理能力。
package websocket

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels/models"
	"icooclaw/pkg/consts"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Manager 负责管理 WebSocket 连接和消息路由。
type Manager struct {
	hub    *Hub
	bus    *bus.MessageBus
	router chi.Router

	// 运行配置
	maxConcurrent int
	authenticate  func(r *http.Request) (string, bool)

	connections atomic.Int64 // 运行状态
	running     atomic.Bool
	logger      *slog.Logger
	upgrader    websocket.Upgrader // WebSocket 升级器
	mu          sync.RWMutex
}

// ManagerConfig 保存管理器配置。
type ManagerConfig struct {
	MaxConcurrent   int
	Authenticate    func(r *http.Request) (string, bool)
	ReadBufferSize  int
	WriteBufferSize int
}

// DefaultManagerConfig 返回默认管理器配置。
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		MaxConcurrent:   100,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
}

// NewManager 创建一个新的 WebSocket 管理器。
func NewManager(cfg *ManagerConfig, logger *slog.Logger, router chi.Router, messageBus *bus.MessageBus) *Manager {
	if cfg == nil {
		cfg = DefaultManagerConfig()
	}
	if logger == nil {
		logger = slog.Default()
	}

	m := &Manager{
		hub:           NewHub(logger),
		bus:           messageBus,
		router:        router,
		maxConcurrent: cfg.MaxConcurrent,
		authenticate:  cfg.Authenticate,
		logger:        logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  cfg.ReadBufferSize,
			WriteBufferSize: cfg.WriteBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
	}

	return m
}

// Start 启动管理器。
func (m *Manager) Start(ctx context.Context) error {
	if m.running.Swap(true) {
		return nil
	}
	if m.router == nil {
		m.running.Store(false)
		return fmt.Errorf("websocket manager requires router")
	}

	m.running.Store(true)
	// 启动 Hub
	go m.hub.Run(ctx)

	// 启动路由
	m.router.Get("/ws", m.HandleWebSocket)
	m.router.Get("/ws/{session_id}", m.HandleWebSocket)
	return nil
}

// Stop 停止管理器。
func (m *Manager) Stop(ctx context.Context) error {
	m.running.Store(false)
	return nil
}

// Send 发送消息到所有连接的客户端。
func (m *Manager) Send(ctx context.Context, msg models.OutboundMessage) error {
	// 通过 session id 获取客户端
	client := m.GetClientFromSession(msg.SessionID)
	if client == nil {
		return nil
	}

	if eventType, payload := buildSendPayload(msg); eventType != "" {
		client.SendJSON(eventType, payload)
		return nil
	}

	client.SendJSON("chunk", map[string]any{
		"data": map[string]any{
			"content": msg.Text,
		},
	})
	client.SendJSON("end", map[string]any{
		"session_id": msg.SessionID,
	})
	return nil
}

// IsRunning 返回管理器是否处于运行状态。
func (m *Manager) IsRunning() bool {
	return m.running.Load()
}

// IsAllowed 检查管理器是否允许指定发送者 ID 发送消息。
func (m *Manager) IsAllowed(senderID string) bool {
	return true
}

// HandleWebSocket 处理 WebSocket 连接升级和管理。
func (m *Manager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session_id")
	if sessionID == "" {
		sessionID = r.URL.Query().Get("session_id")
	}
	// 检查并发连接上限
	if int(m.connections.Load()) >= m.maxConcurrent {
		m.logger.Error("已达到最大并发连接数",
			"max_concurrent", m.maxConcurrent,
			"current_connections", int(m.connections.Load()))
		http.Error(w, "【网关服务】已达到最大并发连接数", http.StatusServiceUnavailable)
		return
	}

	// 如有需要则执行认证
	var userID string
	if m.authenticate != nil {
		var ok bool
		userID, ok = m.authenticate(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	} else {
		userID = "anonymous"
	}

	// 升级连接
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		m.logger.Error("failed to upgrade websocket", "error", err)
		return
	}

	m.connections.Add(1)
	defer m.connections.Add(-1)

	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// 创建客户端，并自动生成会话 ID
	client := NewClient(ClientDependencies{
		Conn:      conn,
		UserID:    userID,
		SessionID: sessionID,
		Manager:   m,
		Logger:    m.logger,
	})

	// 注册到 Hub
	m.hub.Register(client)
	defer m.hub.Unregister(client)

	m.logger.Info("WebSocket客户端连接成功",
		"user_id", userID,
		"client_id", client.ID,
		"session_id", client.sessionID,
		"total_connections", m.connections.Load())

	// 启动客户端
	client.Run(r.Context())
}

// Broadcast 向所有已连接客户端广播消息。
func (m *Manager) Broadcast(message []byte) {
	m.hub.Broadcast(message)
}

// BroadcastTo 向指定客户端发送消息。
func (m *Manager) BroadcastTo(clientID string, message []byte) {
	m.hub.BroadcastTo(clientID, message)
}

// GetClientFromSession 根据会话 ID 获取客户端。
func (m *Manager) GetClientFromSession(sessionID string) *Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, client := range m.hub.clients {
		if client.sessionID == sessionID {
			return client
		}
	}
	return nil
}

// GetQueueStatus 返回当前队列状态。
func (m *Manager) GetQueueStatus() *QueueStatus {
	return &QueueStatus{
		Connections:   int(m.connections.Load()),
		MaxConcurrent: m.maxConcurrent,
	}
}

// SetMaxConcurrent 设置最大并发连接数。
func (m *Manager) SetMaxConcurrent(max int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxConcurrent = max
}

// GetConnectionCount 返回当前连接数。
func (m *Manager) GetConnectionCount() int {
	return int(m.connections.Load())
}

// Run 启动管理器，并启动内部 Hub。
func (m *Manager) Run(ctx context.Context) error {
	m.running.Store(true)
	defer m.running.Store(false)

	m.logger.Info("【WebSocket】管理器已启动")

	// 启动 Hub
	go m.hub.Run(ctx)

	// 等待上下文取消
	<-ctx.Done()

	m.logger.Info("【WebSocket】管理器已停止")
	return ctx.Err()
}

// ProcessMessage 处理收到的聊天消息。
func (m *Manager) ProcessMessage(ctx context.Context, client *Client, msg *ChatMessage) error {
	m.logger.Debug("【WebSocket】处理消息",
		"client_id", client.ID,
		"session_id", msg.SessionID,
		"content_length", len(msg.Content))

	// 发送错误响应消息
	sendErrorResponse := func(errMsg string) {
		client.SendJSON("error", map[string]any{"error": map[string]string{"message": errMsg}})
	}

	// 直接处理消息
	inbound := bus.InboundMessage{
		Channel:   consts.WEBSOCKET,
		SessionID: msg.SessionID,
		Sender:    bus.SenderInfo{ID: client.userID, Name: client.userID},
		Text:      msg.Content,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"agent_id": firstNonEmpty(msg.AgentID, client.agentID),
		},
	}

	err := m.bus.PublishInbound(ctx, inbound)
	if err != nil {
		sendErrorResponse(err.Error())
		return err
	}

	return nil
}

// ProcessStreamMessage 处理流式消息。
func (m *Manager) ProcessStreamMessage(ctx context.Context, client *Client, msg *ChatMessage) error {
	m.logger.Debug("【WebSocket】处理流式消息",
		"client_id", client.ID,
		"session_id", msg.SessionID)

	return m.ProcessMessage(ctx, client, msg)
}

// QueueStatus 表示队列状态。
type QueueStatus struct {
	Connections   int `json:"connections"`
	MaxConcurrent int `json:"max_concurrent"`
}

// ChatMessage 表示收到的聊天消息。
type ChatMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	Stream    bool   `json:"stream,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// ChatResponse 表示聊天响应。
type ChatResponse struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

// StreamEvent 表示流式事件。
type StreamEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id,omitempty"`
	Content   string `json:"content,omitempty"`
	Error     string `json:"error,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func buildSendPayload(msg models.OutboundMessage) (string, map[string]any) {
	eventType := firstNonEmpty(metadataString(msg.Metadata, "event_type"))
	if eventType == "" {
		return "", nil
	}

	payload := map[string]any{
		"session_id": msg.SessionID,
	}

	switch eventType {
	case "chunk":
		data := map[string]any{
			"content": msg.Text,
		}
		if reasoning := metadataString(msg.Metadata, "reasoning"); reasoning != "" {
			data["reasoning"] = reasoning
		}
		if iteration, ok := msg.Metadata["iteration"]; ok {
			data["iteration"] = iteration
		}
		if totalTokens, ok := msg.Metadata["total_tokens"]; ok {
			data["total_tokens"] = totalTokens
		}
		payload["data"] = data
	case "tool_call":
		payload["data"] = map[string]any{
			"tool_call_id": metadataString(msg.Metadata, "tool_call_id"),
			"tool_name":    metadataString(msg.Metadata, "tool_name"),
			"tool_args":    metadataString(msg.Metadata, "tool_args"),
		}
	case "tool_result":
		payload["data"] = map[string]any{
			"tool_call_id": metadataString(msg.Metadata, "tool_call_id"),
			"tool_name":    metadataString(msg.Metadata, "tool_name"),
			"result":       firstNonEmpty(metadataString(msg.Metadata, "result"), msg.Text),
		}
	case "error":
		payload["error"] = map[string]any{
			"message": firstNonEmpty(msg.Text, metadataString(msg.Metadata, "message")),
		}
	case "end":
		if totalTokens, ok := msg.Metadata["total_tokens"]; ok {
			payload["data"] = map[string]any{
				"total_tokens": totalTokens,
			}
		}
	default:
		return "", nil
	}

	return eventType, payload
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
