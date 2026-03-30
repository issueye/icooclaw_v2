// Package dingtalk provides DingTalk channel implementation for icooclaw.
package dingtalk

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/client"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels/models"
	"icooclaw/pkg/consts"
	errs "icooclaw/pkg/errors"
	"icooclaw/pkg/utils"
)

// Channel 钉钉渠道实现.
type Channel struct {
	name         string // channel name from database
	channelType  string // channel type from database (e.g., "dingtalk" or "钉钉")
	config       Config
	bus          *bus.MessageBus
	logger       *slog.Logger
	clientID     string
	clientSecret string
	streamClient *client.StreamClient
	apiClient    *APIClient // API client for non-stream operations
	ctx          context.Context
	cancel       context.CancelFunc

	// Map to store session webhooks for each chat
	sessionWebhooks sync.Map // chatID -> sessionWebhook
	running         atomic.Bool
}

// New 创建钉钉渠道实例.
func New(logger *slog.Logger, bus *bus.MessageBus, cfgStr string) (models.Channel, error) {
	c := &Channel{
		name:        consts.DINGTALK,
		channelType: consts.DINGTALK,
		bus:         bus,
		logger:      logger,
	}

	// 解析配置
	cfgMap := utils.ParseConfig(cfgStr)
	cfg, err := ParseConfig(cfgMap)
	if err != nil {
		return nil, err
	}

	// 校验配置
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("钉钉client_id和client_secret不能为空")
	}

	c.clientID = cfg.ClientID
	c.clientSecret = cfg.ClientSecret
	c.config = cfg
	c.apiClient = NewAPIClient(cfg.ClientID, cfg.ClientSecret, logger)

	return c, nil
}

// Start 启动钉钉渠道实例.
func (c *Channel) Start(ctx context.Context) error {
	c.logger.With("name", "【钉钉】").Info("启动通道...")

	c.ctx, c.cancel = context.WithCancel(ctx)

	// Create credential config
	cred := client.NewAppCredentialConfig(c.clientID, c.clientSecret)

	// Create the stream client with options
	c.streamClient = client.NewStreamClient(
		client.WithAppCredential(cred),
		client.WithAutoReconnect(true),
	)

	// Register chatbot callback handler
	c.streamClient.RegisterChatBotCallbackRouter(c.onChatBotMessageReceived)

	// Start the stream client
	if err := c.streamClient.Start(c.ctx); err != nil {
		c.logger.With("name", "【钉钉】").Error("启动通道失败", "error", err)
		return fmt.Errorf("启动通道失败：%w", err)
	}

	c.running.Store(true)
	c.logger.With("name", "【钉钉】").Info("通道已启动（流模式）")
	return nil
}

// Stop 停止钉钉渠道实例.
func (c *Channel) Stop(ctx context.Context) error {
	c.logger.With("name", "【钉钉】").Info("关闭通道...")

	if c.cancel != nil {
		c.cancel()
	}

	if c.streamClient != nil {
		c.streamClient.Close()
	}

	c.running.Store(false)
	c.logger.With("name", "【钉钉】").Info("通道已停止")
	return nil
}

// IsRunning 检查渠道是否正在运行.
func (c *Channel) IsRunning() bool {
	return c.running.Load()
}

// IsAllowed 检查发送者是否被允许.
func (c *Channel) IsAllowed(senderID string) bool {
	return models.IsSenderAllowed(c.config.AllowFrom, senderID)
}

// IsAllowedSender 检查发送者是否被允许.
func (c *Channel) IsAllowedSender(sender models.SenderInfo) bool {
	return c.IsAllowed(sender.ID)
}

// ReasoningChannelID 获取推理消息目标会话 ID.
func (c *Channel) ReasoningChannelID() string {
	return c.config.ReasoningChatID
}

// Send 发送消息到钉钉渠道.
func (c *Channel) Send(ctx context.Context, msg models.OutboundMessage) error {
	if !c.IsRunning() {
		return errs.ErrNotRunning
	}

	// Try session webhook first (for stream-mode replies)
	if sessionWebhookRaw, ok := c.sessionWebhooks.Load(msg.SessionID); ok {
		if sessionWebhook, ok := sessionWebhookRaw.(string); ok {
			c.logger.With("name", "【钉钉】").Debug("发送消息", "session_id", msg.SessionID, "preview", truncate(msg.Text, 100))
			return c.SendDirectReply(ctx, sessionWebhook, msg.Text)
		}
	}

	// Fallback: use API client to send message directly
	if c.apiClient != nil && c.config.AgentID != 0 {
		c.logger.With("name", "【钉钉】").Debug("通过API发送消息", "session_id", msg.SessionID, "preview", truncate(msg.Text, 100))
		content := BuildMarkdownContent("AI Assistant", msg.Text)
		return c.apiClient.SendMessage(ctx, c.config.AgentID, msg.SessionID, string(MessageTypeMarkdown), content)
	}

	c.logger.With("name", "【钉钉】").Error("无法发送消息：无会话webhook且API客户端未配置", "session_id", msg.SessionID)
	return fmt.Errorf("未找到会话webhook且API客户端未配置，无法发送消息：%s", msg.SessionID)
}

// onChatBotMessageReceived 处理钉钉渠道的回调消息.
func (c *Channel) onChatBotMessageReceived(
	ctx context.Context,
	data *chatbot.BotCallbackDataModel,
) ([]byte, error) {
	// Process message asynchronously to avoid blocking the DingTalk SDK
	go c.processDingTalkMessage(ctx, data)
	return nil, nil
}

// processDingTalkMessage 处理钉钉渠道的回调消息.
func (c *Channel) processDingTalkMessage(_ context.Context, data *chatbot.BotCallbackDataModel) {
	// Extract message content from Text field
	content := data.Text.Content
	if content == "" {
		// Try to extract from Content interface{} if Text is empty
		if contentMap, ok := data.Content.(map[string]any); ok {
			if textContent, ok := contentMap["content"].(string); ok {
				content = textContent
			}
		}
	}

	if content == "" {
		return // Ignore empty messages
	}

	senderID := data.SenderStaffId
	senderNick := data.SenderNick
	chatID := senderID
	if data.ConversationType != "1" {
		// For group chats
		chatID = data.ConversationId
	}

	// Store the session webhook for this chat so we can reply later
	c.sessionWebhooks.Store(chatID, data.SessionWebhook)

	// Check allowlist
	if !c.IsAllowed(senderID) {
		return
	}

	metadata := map[string]any{
		"sender_name":       senderNick,
		"conversation_id":   data.ConversationId,
		"conversation_type": data.ConversationType,
		"platform":          "dingtalk",
		"session_webhook":   data.SessionWebhook,
	}

	c.logger.With("name", "【钉钉】").Debug("收到消息",
		"sender_nick", senderNick,
		"sender_id", senderID,
		"preview", truncate(content, 50),
	)

	// Build inbound message
	inboundMsg := bus.InboundMessage{
		Channel:   consts.DINGTALK,
		SessionID: chatID,
		Sender:    bus.SenderInfo{ID: senderID, Name: senderNick},
		Text:      content,
		Metadata:  metadata,
	}

	// Publish to bus with timeout to avoid indefinite blocking
	pubCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.bus.PublishInbound(pubCtx, inboundMsg); err != nil {
		c.logger.With("name", "【钉钉】").Error("发布消息失败", "error", err)
	}
}

// SendDirectReply 发送直接回复到会话.
func (c *Channel) SendDirectReply(ctx context.Context, sessionWebhook, content string) error {
	replier := chatbot.NewChatbotReplier()

	// Convert string content to []byte for the API
	contentBytes := []byte(content)
	titleBytes := []byte("AI Assistant")

	// Send markdown formatted reply
	err := replier.SimpleReplyMarkdown(
		ctx,
		sessionWebhook,
		titleBytes,
		contentBytes,
	)
	if err != nil {
		c.logger.With("name", "【钉钉】").Error("发送失败", "error", err)
		return fmt.Errorf("dingtalk send: %w", errs.ErrTemporary)
	}

	return nil
}

func truncate(s string, maxLen int) string {
	return utils.Truncate(s, maxLen)
}

// ParseConfig 解析钉钉渠道配置.
func ParseConfig(config map[string]any) (Config, error) {
	cfg := Config{}

	if v, ok := config["enabled"]; ok {
		if b, ok := v.(bool); ok {
			cfg.Enabled = b
		}
	}

	if v, ok := config["client_id"]; ok {
		if s, ok := v.(string); ok {
			cfg.ClientID = s
		}
	}

	if v, ok := config["client_secret"]; ok {
		if s, ok := v.(string); ok {
			cfg.ClientSecret = s
		}
	}

	if v, ok := config["agent_id"]; ok {
		switch val := v.(type) {
		case int64:
			cfg.AgentID = val
		case int:
			cfg.AgentID = int64(val)
		case float64:
			cfg.AgentID = int64(val)
		}
	}

	if v, ok := config["allow_from"]; ok {
		if arr, ok := v.([]any); ok {
			for _, item := range arr {
				if s, ok := item.(string); ok {
					cfg.AllowFrom = append(cfg.AllowFrom, s)
				}
			}
		}
	}

	if v, ok := config["reasoning_chat_id"]; ok {
		if s, ok := v.(string); ok {
			cfg.ReasoningChatID = s
		}
	}

	return cfg, nil
}
