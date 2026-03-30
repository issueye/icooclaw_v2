// Package feishu provides Feishu/Lark channel implementation for icooclaw.
package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdispatcher "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels/models"
	"icooclaw/pkg/consts"
	errs "icooclaw/pkg/errors"
	"icooclaw/pkg/utils"
)

// Channel channels 实现飞书渠道的通信逻辑.
type Channel struct {
	name        string // channel name from database
	channelType string // channel type from database (e.g., "feishu" or "飞书")
	config      Config
	bus         *bus.MessageBus
	client      *lark.Client
	wsClient    *larkws.Client
	logger      *slog.Logger

	botOpenID atomic.Value // stores string; populated lazily for @mention detection

	running atomic.Bool
	mu      sync.Mutex
	cancel  context.CancelFunc
}

// New 创建飞书渠道实例.
func New(logger *slog.Logger, bus *bus.MessageBus, cfgStr string) (models.Channel, error) {
	c := &Channel{
		channelType: consts.FEISHU_CN,
		bus:         bus,
		logger:      logger,
	}

	// 解析配置
	cfgMap := utils.ParseConfig(cfgStr)
	cfg, err := ParseConfig(cfgMap)
	if err != nil {
		return nil, err
	}

	// 验证配置
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return nil, fmt.Errorf("飞书渠道配置中app_id和app_secret不能为空。")
	}

	// 创建飞书客户端
	c.client = lark.NewClient(cfg.AppID, cfg.AppSecret)
	return c, nil
}

// Start 启动飞书渠道实例.
func (c *Channel) Start(ctx context.Context) error {
	if c.config.AppID == "" || c.config.AppSecret == "" {
		return fmt.Errorf("feishu app_id or app_secret is empty")
	}

	// 标记为运行中
	c.running.Store(true)

	// Fetch bot open_id via API for reliable @mention detection.
	// Use a detached context with timeout to avoid cancellation from parent context.
	fetchCtx, fetchCancel := context.WithTimeout(context.Background(), 30*time.Second)
	err := c.fetchBotOpenID(fetchCtx)
	fetchCancel()
	if err != nil {
		c.logger.With("name", "【飞书】").Error("获取机器人open_id失败", "error", err)
		return fmt.Errorf("获取机器人open_id失败：%w", err)
	}

	dispatcher := larkdispatcher.NewEventDispatcher(c.config.VerificationToken, c.config.EncryptKey).
		OnP2MessageReceiveV1(c.handleMessageReceive)

	runCtx, cancel := context.WithCancel(ctx)

	c.mu.Lock()
	c.cancel = cancel
	c.wsClient = larkws.NewClient(
		c.config.AppID,
		c.config.AppSecret,
		larkws.WithEventHandler(dispatcher),
	)
	wsClient := c.wsClient
	c.mu.Unlock()

	c.running.Store(true)
	c.logger.With("name", "【飞书】").Info("启动通道...（流模式）")

	go func() {
		if err := wsClient.Start(runCtx); err != nil {
			c.logger.With("name", "【飞书】").Error("启动通道失败：", "error", err)
		}
	}()

	return nil
}

// Stop stops the Feishu channel.
func (c *Channel) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.wsClient = nil
	c.mu.Unlock()

	c.running.Store(false)
	c.logger.With("name", "【飞书】").Info("通道已停止")
	return nil
}

// IsRunning returns true if the channel is running.
func (c *Channel) IsRunning() bool {
	return c.running.Load()
}

// IsAllowed checks if a sender is allowed.
func (c *Channel) IsAllowed(senderID string) bool {
	return models.IsSenderAllowed(c.config.AllowFrom, senderID)
}

// IsAllowedSender checks if a sender is allowed (with full info).
func (c *Channel) IsAllowedSender(sender models.SenderInfo) bool {
	return c.IsAllowed(sender.ID)
}

// ReasoningChannelID returns the channel ID for reasoning messages.
func (c *Channel) ReasoningChannelID() string {
	return c.config.ReasoningChatID
}

// Send sends a message using Interactive Card format for markdown rendering.
func (c *Channel) Send(ctx context.Context, msg models.OutboundMessage) error {
	if !c.IsRunning() {
		return errs.ErrNotRunning
	}

	if msg.SessionID == "" {
		c.logger.With("name", "【飞书】").Error("发送消息失败：sessionID不能为空", "error", errs.ErrSendFailed)
		return fmt.Errorf("session ID is empty: %w", errs.ErrSendFailed)
	}

	// Build interactive card with markdown content
	cardContent, err := buildMarkdownCard(msg.Text)
	if err != nil {
		c.logger.With("name", "【飞书】").Error("发送消息失败：卡片构建失败", "error", err)
		return fmt.Errorf("feishu send: card build failed: %w", err)
	}
	return c.sendCard(ctx, msg.SessionID, cardContent)
}

// EditMessage implements channels.MessageEditor.
func (c *Channel) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	cardContent, err := buildMarkdownCard(content)
	if err != nil {
		return fmt.Errorf("feishu edit: card build failed: %w", err)
	}

	req := larkim.NewPatchMessageReqBuilder().
		MessageId(messageID).
		Body(larkim.NewPatchMessageReqBodyBuilder().Content(cardContent).Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Patch(ctx, req)
	if err != nil {
		return fmt.Errorf("feishu edit: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("feishu edit api error (code=%d msg=%s)", resp.Code, resp.Msg)
	}
	return nil
}

// SendPlaceholder implements channels.PlaceholderCapable.
func (c *Channel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	text := "Thinking..."

	cardContent, err := buildMarkdownCard(text)
	if err != nil {
		return "", fmt.Errorf("feishu placeholder: card build failed: %w", err)
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeInteractive).
			Content(cardContent).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("feishu placeholder send: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("feishu placeholder api error (code=%d msg=%s)", resp.Code, resp.Msg)
	}

	if resp.Data != nil && resp.Data.MessageId != nil {
		return *resp.Data.MessageId, nil
	}
	return "", nil
}

// ReactToMessage implements channels.ReactionCapable.
func (c *Channel) ReactToMessage(ctx context.Context, chatID, messageID string) (func(), error) {
	req := larkim.NewCreateMessageReactionReqBuilder().
		MessageId(messageID).
		Body(larkim.NewCreateMessageReactionReqBodyBuilder().
			ReactionType(larkim.NewEmojiBuilder().EmojiType("Pin").Build()).
			Build()).
		Build()

	resp, err := c.client.Im.V1.MessageReaction.Create(ctx, req)
	if err != nil {
		c.logger.With("name", "【飞书】").Error("发送消息失败：添加反应失败", "error", err)
		return func() {}, fmt.Errorf("feishu react: %w", err)
	}
	if !resp.Success() {
		c.logger.With("name", "【飞书】").Error("发送消息失败：添加反应失败", slog.Any("code", resp.Code), slog.Any("msg", resp.Msg))
		return func() {}, fmt.Errorf("feishu react: %w", err)
	}

	var reactionID string
	if resp.Data != nil && resp.Data.ReactionId != nil {
		reactionID = *resp.Data.ReactionId
	}
	if reactionID == "" {
		return func() {}, nil
	}

	var undone atomic.Bool
	undo := func() {
		if !undone.CompareAndSwap(false, true) {
			return
		}
		delReq := larkim.NewDeleteMessageReactionReqBuilder().
			MessageId(messageID).
			ReactionId(reactionID).
			Build()
		_, _ = c.client.Im.V1.MessageReaction.Delete(context.Background(), delReq)
	}
	return undo, nil
}

// --- Inbound message handling ---

func (c *Channel) handleMessageReceive(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return nil
	}

	// Process message asynchronously to avoid blocking the Feishu SDK
	go c.processMessageAsync(ctx, event)
	return nil
}

// processMessageAsync processes an incoming message asynchronously.
func (c *Channel) processMessageAsync(ctx context.Context, event *larkim.P2MessageReceiveV1) {
	message := event.Event.Message
	sender := event.Event.Sender

	// 打印消息
	c.logger.With("name", "【飞书】").Info("接收到飞书消息",
		slog.Any("message", message),
		slog.Any("sender", sender),
		slog.Any("chat_id", message.ChatId),
	)

	chatID := stringValue(message.ChatId)
	if chatID == "" {
		return
	}

	senderID := extractSenderID(sender)
	if senderID == "" {
		senderID = "unknown"
	}

	// Filter out messages sent by the bot itself to avoid loops
	if botOpenID, ok := c.botOpenID.Load().(string); ok && botOpenID != "" {
		if senderID == botOpenID {
			c.logger.With("name", "【飞书】").Debug("忽略机器人自己发送的消息")
			return
		}
	}

	messageType := stringValue(message.MessageType)
	messageID := stringValue(message.MessageId)
	rawContent := stringValue(message.Content)

	// Check allowlist early
	if !c.IsAllowed(senderID) {
		return
	}

	// Extract content based on message type
	content := extractContent(messageType, rawContent)

	// Handle media messages
	var mediaRefs []string
	if messageID != "" {
		mediaRefs = c.downloadInboundMedia(ctx, chatID, messageID, messageType, rawContent)
	}

	// Append media tags to content
	content = appendMediaTags(content, messageType, mediaRefs)

	if content == "" {
		content = "[empty message]"
	}

	metadata := map[string]any{}
	if messageID != "" {
		metadata["message_id"] = messageID
	}
	if messageType != "" {
		metadata["message_type"] = messageType
	}
	chatType := stringValue(message.ChatType)
	if chatType != "" {
		metadata["chat_type"] = chatType
	}
	if sender != nil && sender.TenantKey != nil {
		metadata["tenant_key"] = *sender.TenantKey
	}

	// Build inbound message
	// 构建入站消息
	inboundMsg := bus.InboundMessage{
		Channel:   consts.FEISHU_CN,
		SessionID: chatID,
		Sender:    bus.SenderInfo{ID: senderID},
		Text:      content,
		Media:     mediaRefs,
		Metadata:  metadata,
	}

	c.logger.With("name", "【飞书】").Info("收到消息",
		slog.String("sender_id", senderID),
		"chat_id", chatID,
		"message_id", messageID,
		"preview", truncate(content, 80),
	)

	// Publish to bus with timeout to avoid indefinite blocking
	pubCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.bus.PublishInbound(pubCtx, inboundMsg); err != nil {
		c.logger.With("name", "【飞书】").Error("发布消息失败", "error", err)
	}
}

// --- Internal helpers ---

// fetchBotOpenID calls the Feishu bot info API to retrieve and store the bot's open_id.
func (c *Channel) fetchBotOpenID(ctx context.Context) error {
	resp, err := c.client.Do(ctx, &larkcore.ApiReq{
		HttpMethod:                http.MethodGet,
		ApiPath:                   "/open-apis/bot/v3/info",
		SupportedAccessTokenTypes: []larkcore.AccessTokenType{larkcore.AccessTokenTypeTenant},
	})
	if err != nil {
		c.logger.With("name", "【飞书】").Error("获取机器人open_id失败", "error", err)
		return fmt.Errorf("bot info request: %w", err)
	}

	var result struct {
		Code int `json:"code"`
		Bot  struct {
			OpenID string `json:"open_id"`
		} `json:"bot"`
	}
	if err := json.Unmarshal(resp.RawBody, &result); err != nil {
		c.logger.With("name", "【飞书】").Error("解析机器人open_id失败", "error", err)
		return fmt.Errorf("机器人信息解析失败 %w", err)
	}
	if result.Code != 0 {
		c.logger.With("name", "【飞书】").Error("获取机器人open_id失败", slog.Any("code", result.Code))
		return fmt.Errorf("机器人信息获取失败，错误码: %d", result.Code)
	}
	if result.Bot.OpenID == "" {
		c.logger.With("name", "【飞书】").Error("获取机器人open_id失败", "error", "open_id为空")
		return fmt.Errorf("机器人open_id为空")
	}

	c.botOpenID.Store(result.Bot.OpenID)
	c.logger.With("name", "【飞书】").Info("获取机器人open_id成功", slog.Any("open_id", result.Bot.OpenID))
	return nil
}

// sendCard sends an interactive card message to a chat.
func (c *Channel) sendCard(ctx context.Context, chatID, cardContent string) error {
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeInteractive).
			Content(cardContent).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		c.logger.With("name", "【飞书】").Error("发送消息失败：发送卡片失败", "error", err)
		return fmt.Errorf("发送卡片失败 %w", err)
	}

	if !resp.Success() {
		c.logger.With("name", "【飞书】").Error("发送消息失败：发送卡片失败", slog.Any("code", resp.Code), slog.Any("msg", resp.Msg))
		return fmt.Errorf("发送卡片失败 (code=%d msg=%s)", resp.Code, resp.Msg)
	}

	c.logger.With("name", "【飞书】").Info("发送卡片成功", slog.String("chat_id", chatID))
	return nil
}

// downloadInboundMedia downloads media from inbound messages.
func (c *Channel) downloadInboundMedia(
	ctx context.Context,
	chatID, messageID, messageType, rawContent string,
) []string {
	var refs []string

	switch messageType {
	case larkim.MsgTypeImage:
		imageKey := extractImageKey(rawContent)
		if imageKey == "" {
			return nil
		}
		ref := c.downloadResource(ctx, messageID, imageKey, "image", ".jpg")
		if ref != "" {
			refs = append(refs, ref)
		}

	case larkim.MsgTypeFile, larkim.MsgTypeAudio, larkim.MsgTypeMedia:
		fileKey := extractFileKey(rawContent)
		if fileKey == "" {
			return nil
		}
		var ext string
		switch messageType {
		case larkim.MsgTypeAudio:
			ext = ".ogg"
		case larkim.MsgTypeMedia:
			ext = ".mp4"
		default:
			ext = ""
		}
		ref := c.downloadResource(ctx, messageID, fileKey, "file", ext)
		if ref != "" {
			refs = append(refs, ref)
		}
	}

	return refs
}

// downloadResource downloads a message resource from Feishu.
func (c *Channel) downloadResource(
	ctx context.Context,
	messageID, fileKey, resourceType, fallbackExt string,
) string {
	req := larkim.NewGetMessageResourceReqBuilder().
		MessageId(messageID).
		FileKey(fileKey).
		Type(resourceType).
		Build()

	resp, err := c.client.Im.V1.MessageResource.Get(ctx, req)
	if err != nil {
		c.logger.With("name", "【飞书】").Error("下载资源失败：", "error", err)
		return ""
	}
	if !resp.Success() {
		c.logger.With("name", "【飞书】").Error("下载资源失败：", slog.Any("code", resp.Code), slog.Any("msg", resp.Msg))
		return ""
	}

	if resp.File == nil {
		return ""
	}
	// Safely close the underlying reader if it implements io.Closer
	if closer, ok := resp.File.(io.Closer); ok {
		defer closer.Close()
	}

	filename := resp.FileName
	if filename == "" {
		filename = fileKey
	}
	if filepath.Ext(filename) == "" && fallbackExt != "" {
		filename += fallbackExt
	}

	// Write to temp directory
	mediaDir := filepath.Join(os.TempDir(), "icooclaw_media")
	if mkdirErr := os.MkdirAll(mediaDir, 0o700); mkdirErr != nil {
		c.logger.With("name", "【飞书】").Error("创建媒体目录失败", slog.String("目录", mediaDir), "error", mkdirErr.Error())
		return ""
	}
	ext := filepath.Ext(filename)
	localPath := filepath.Join(mediaDir, sanitizeFilename(messageID+"-"+fileKey+ext))

	out, err := os.Create(localPath)
	if err != nil {
		c.logger.With("name", "【飞书】").Error("创建媒体文件失败：", "error", err)
		return ""
	}

	if _, copyErr := io.Copy(out, resp.File); copyErr != nil {
		out.Close()
		os.Remove(localPath)
		c.logger.With("name", "【飞书】").Error("下载资源失败：", "error", copyErr.Error())
		return ""
	}
	out.Close()

	return localPath
}

// ParseConfig 解析飞书渠道配置.
func ParseConfig(config map[string]any) (Config, error) {
	cfg := Config{}

	if v, ok := config["enabled"]; ok {
		if b, ok := v.(bool); ok {
			cfg.Enabled = b
		}
	}

	if v, ok := config["app_id"]; ok {
		if s, ok := v.(string); ok {
			cfg.AppID = s
		}
	}

	if v, ok := config["app_secret"]; ok {
		if s, ok := v.(string); ok {
			cfg.AppSecret = s
		}
	}

	if v, ok := config["encrypt_key"]; ok {
		if s, ok := v.(string); ok {
			cfg.EncryptKey = s
		}
	}

	if v, ok := config["verification_token"]; ok {
		if s, ok := v.(string); ok {
			cfg.VerificationToken = s
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

// ParseConfigFromJSON 从 JSON 字节数组解析飞书渠道配置.
func ParseConfigFromJSON(data []byte) (Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
