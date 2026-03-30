package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Client 表示一个 WebSocket 客户端连接。
type Client struct {
	ID        string
	conn      *websocket.Conn
	send      chan []byte
	userID    string
	sessionID string
	agentID   string

	manager *Manager
	logger  *slog.Logger

	// 运行状态
	connected  atomic.Bool
	lastPing   time.Time
	lastPong   time.Time
	messageSeq atomic.Uint64

	// 运行配置
	writeWait      time.Duration
	pongWait       time.Duration
	pingPeriod     time.Duration
	maxMessageSize int64

	mu sync.Mutex
}

// ClientConfig 保存客户端配置。
type ClientConfig struct {
	WriteWait      time.Duration
	PongWait       time.Duration
	PingPeriod     time.Duration
	MaxMessageSize int64
	SendBufferSize int
}

type ClientDependencies struct {
	Conn      *websocket.Conn
	UserID    string
	SessionID string
	AgentID   string
	Manager   *Manager
	Logger    *slog.Logger
	Config    *ClientConfig
}

// DefaultClientConfig 返回默认客户端配置。
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		WriteWait:      10 * time.Second,
		PongWait:       60 * time.Second,
		PingPeriod:     (60 * time.Second * 9) / 10,
		MaxMessageSize: 512 * 1024, // 512KB
		SendBufferSize: 256,
	}
}

// NewClient 创建一个新的 WebSocket 客户端。
func NewClient(deps ClientDependencies) *Client {
	cfg := deps.Config
	if cfg == nil {
		cfg = DefaultClientConfig()
	}
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}

	return &Client{
		ID:             uuid.New().String(),
		conn:           deps.Conn,
		send:           make(chan []byte, cfg.SendBufferSize),
		userID:         deps.UserID,
		sessionID:      deps.SessionID,
		agentID:        deps.AgentID,
		manager:        deps.Manager,
		logger:         deps.Logger,
		writeWait:      cfg.WriteWait,
		pongWait:       cfg.PongWait,
		pingPeriod:     cfg.PingPeriod,
		maxMessageSize: cfg.MaxMessageSize,
	}
}

// Run 启动客户端的读写循环。
func (c *Client) Run(ctx context.Context) {
	c.connected.Store(true)
	defer c.connected.Store(false)
	defer c.Close()

	// 启动写循环
	go c.writePump(ctx)

	// 启动读循环
	c.readPump(ctx)
}

// readPump 从 WebSocket 连接持续读取消息并交给上层处理。
func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.connected.Store(false)
	}()

	// 设置读取限制和超时
	c.conn.SetReadLimit(c.maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(c.pongWait))

	// 设置 pong 处理器
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.pongWait))
		c.lastPong = time.Now()
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
			messageType, message, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					c.logger.Error("unexpected close error", "error", err, "client_id", c.ID)
				}
				return
			}

			c.lastPing = time.Now()
			c.messageSeq.Add(1)

			// 处理消息
			c.handleMessage(ctx, messageType, message)
		}
	}
}

// writePump 将 Hub 中的消息持续写入 WebSocket 连接。
func (c *Client) writePump(ctx context.Context) {
	ticker := time.NewTicker(c.pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			// 优雅关闭连接
			c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return

		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
			if !ok {
				// Hub 已关闭该通道
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 批量写出排队消息
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 处理收到的消息。
func (c *Client) handleMessage(ctx context.Context, messageType int, message []byte) {
	if messageType != websocket.TextMessage {
		c.logger.Warn("消息类型错误，仅支持文本消息", "type", messageType, "client_id", c.ID)
		c.SendError("消息类型错误")
		return
	}

	fmt.Println("收到消息", string(message))

	// 解析消息
	var msg ChatMessage
	err := json.Unmarshal(message, &msg)
	if err != nil {
		c.logger.Error("解析消息失败", "error", err, "client_id", c.ID)
		c.SendError("消息格式错误")
		return
	}

	// 如果未提供时间戳，则补当前时间
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().Unix()
	}

	// 按消息类型分发处理
	switch msg.Type {
	case "chat", "":
		// 校验消息
		if msg.Content == "" {
			c.SendError("消息内容不能为空")
			return
		}
		// 如果未提供会话ID，则使用当前会话ID
		if msg.SessionID == "" {
			msg.SessionID = c.sessionID
			c.SendError("会话ID不能为空")
			return
		}

		if c.manager == nil {
			c.SendError("管理器未配置")
			return
		}

		// 处理流式消息
		if msg.Stream {
			go c.manager.ProcessStreamMessage(ctx, c, &msg)
		} else {
			go c.manager.ProcessMessage(ctx, c, &msg)
		}

	case "ping":
		c.SendJSON("pong", nil)

	default:
		c.logger.Warn("unknown message type", "type", msg.Type, "client_id", c.ID)
		c.SendError("unknown message type: " + msg.Type)
	}
}

// Send 将消息放入发送队列。
func (c *Client) Send(message []byte) bool {
	if !c.connected.Load() {
		return false
	}

	select {
	case c.send <- message:
		return true
	default:
		c.logger.Warn("发送消息队列已满，丢弃消息", "client_id", c.ID)
		return false
	}
}

// SendJSON 向客户端发送 JSON 消息。
func (c *Client) SendJSON(typeStr string, msg map[string]any) bool {
	if msg == nil {
		msg = make(map[string]any)
	}

	// 如果未提供时间戳，则补当前时间
	if msg["timestamp"] == nil {
		msg["timestamp"] = time.Now().Unix()
	}

	msg["type"] = typeStr

	data, err := json.Marshal(msg)
	if err != nil {
		c.logger.Error("failed to marshal json", "error", err, "client_id", c.ID)
		return false
	}
	return c.Send(data)
}

// SendError 向客户端发送错误消息。
func (c *Client) SendError(message string) {
	c.SendJSON("error", map[string]any{"message": message})
}

// Close 关闭客户端连接。
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected.Load() {
		return nil
	}

	c.connected.Store(false)
	close(c.send)
	return c.conn.Close()
}

// IsConnected 返回客户端是否已连接。
func (c *Client) IsConnected() bool {
	return c.connected.Load()
}

// GetStats 返回客户端统计信息。
func (c *Client) GetStats() *ClientStats {
	return &ClientStats{
		ID:         c.ID,
		UserID:     c.userID,
		SessionID:  c.sessionID,
		AgentID:    c.agentID,
		Connected:  c.connected.Load(),
		MessageSeq: c.messageSeq.Load(),
		LastPing:   c.lastPing,
		LastPong:   c.lastPong,
	}
}

// ClientStats 表示客户端统计信息。
type ClientStats struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	SessionID  string    `json:"session_id"`
	AgentID    string    `json:"agent_id,omitempty"`
	Connected  bool      `json:"connected"`
	MessageSeq uint64    `json:"message_seq"`
	LastPing   time.Time `json:"last_ping"`
	LastPong   time.Time `json:"last_pong"`
}
