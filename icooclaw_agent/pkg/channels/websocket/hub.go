package websocket

import (
	"context"
	"log/slog"
	"sync"
)

// Hub 维护活动客户端集并广播消息。
type Hub struct {
	clients    map[string]*Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client

	// Per-client channels for targeted messages
	clientChannels map[string]chan []byte

	logger *slog.Logger

	mu sync.RWMutex
}

// NewHub 创建一个新的Hub。
func NewHub(logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}

	return &Hub{
		clients:        make(map[string]*Client),
		broadcast:      make(chan []byte, 256),
		register:       make(chan *Client, 16),
		unregister:     make(chan *Client, 16),
		clientChannels: make(map[string]chan []byte),
		logger:         logger,
	}
}

// Run 启动Hub。
func (h *Hub) Run(ctx context.Context) {
	h.logger.Debug("Hub 启动")

	for {
		select {
		case <-ctx.Done():
			h.logger.Debug("Hub 停止")
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.clientChannels[client.ID] = make(chan []byte, 64)
			h.mu.Unlock()

			h.logger.Debug("客户端注册", "client_id", client.ID, "total", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				if ch, ok := h.clientChannels[client.ID]; ok {
					close(ch)
					delete(h.clientChannels, client.ID)
				}
			}
			h.mu.Unlock()

			h.logger.Debug("客户端注销", "client_id", client.ID, "total", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client buffer full, skip
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register 注册客户端。
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister 注销客户端。
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast 广播给所有已连接客户端。
func (h *Hub) Broadcast(message []byte) {
	select {
	case h.broadcast <- message:
	default:
		h.logger.Warn("broadcast channel full, dropping message")
	}
}

// BroadcastTo 广播给指定客户端。
func (h *Hub) BroadcastTo(clientID string, message []byte) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	client, ok := h.clients[clientID]
	if !ok {
		return false
	}

	select {
	case client.send <- message:
		return true
	default:
		return false
	}
}

// BroadcastToUser 广播给指定用户的客户端。
func (h *Hub) BroadcastToUser(userID string, message []byte) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, client := range h.clients {
		if client.userID == userID {
			select {
			case client.send <- message:
				count++
			default:
				// Client buffer full, skip
			}
		}
	}
	return count
}

// GetClient 获取客户端。
func (h *Hub) GetClient(clientID string) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	client, ok := h.clients[clientID]
	return client, ok
}

// GetClientsByUser 获取指定用户的客户端。
func (h *Hub) GetClientsByUser(userID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var clients []*Client
	for _, client := range h.clients {
		if client.userID == userID {
			clients = append(clients, client)
		}
	}
	return clients
}

// GetClientCount 获取已连接客户端数量。
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetClientIDs 获取所有已连接客户端ID。
func (h *Hub) GetClientIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	ids := make([]string, 0, len(h.clients))
	for id := range h.clients {
		ids = append(ids, id)
	}
	return ids
}

// GetClientStats 获取所有客户端的统计信息。
func (h *Hub) GetClientStats() []*ClientStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := make([]*ClientStats, 0, len(h.clients))
	for _, client := range h.clients {
		stats = append(stats, client.GetStats())
	}
	return stats
}
