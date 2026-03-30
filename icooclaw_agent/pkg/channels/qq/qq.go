package qq

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/event"
	"github.com/tencent-connect/botgo/openapi"
	"github.com/tencent-connect/botgo/token"
	"golang.org/x/oauth2"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels/errs"
	"icooclaw/pkg/channels/models"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/utils"
)

const (
	dedupTTL          = 5 * time.Minute
	dedupInterval     = 60 * time.Second
	dedupMaxSize      = 10000
	mediaDedupTTL     = 10 * time.Minute
	mediaDedupMaxSize = 5000
	typingResend      = 8 * time.Second
	typingSeconds     = 10
)

// Channel QQ 渠道
type Channel struct {
	name           string
	channelType    string
	config         Config
	bus            *bus.MessageBus
	api            openapi.OpenAPI
	tokenSource    oauth2.TokenSource
	logger         *logHandler
	ctx            context.Context
	cancel         context.CancelFunc
	sessionManager botgo.SessionManager

	chatType       sync.Map
	lastMsgID      sync.Map
	msgSeqCounters sync.Map

	dedup      map[string]time.Time
	mediaDedup map[string]time.Time
	muDedup    sync.Mutex
	done       chan struct{}
	stopOnce   sync.Once
	running    atomic.Bool
	mu         sync.Mutex
}

// New 创建一个新的 QQ 渠道实例
func New(logger *slog.Logger, bus *bus.MessageBus, cfgStr string) (models.Channel, error) {
	// 创建 QQ 渠道实例
	ch := &Channel{
		name:        consts.QQ,
		channelType: consts.QQ,
		config:      Config{},
		bus:         bus,
		logger:      NewLogHandler(logger, "【QQ】"),
		dedup:       make(map[string]time.Time),
		mediaDedup:  make(map[string]time.Time),
		done:        make(chan struct{}),
	}

	logger.Info("QQ渠道配置信息", slog.Any("config", cfgStr))

	// 解析配置
	cfgMap := utils.ParseConfig(cfgStr)
	cfg, err := ParseConfig(cfgMap)
	if err != nil {
		return nil, err
	}

	ch.config = cfg
	return ch, nil
}

func (c *Channel) Start(ctx context.Context) error {
	if c.config.AppID == "" || c.config.AppSecret == "" {
		return fmt.Errorf("qq app_id and app_secret are required")
	}

	// 标记为运行中
	c.running.Store(true)
	botgo.SetLogger(c.logger)
	c.logger.Info("启动 QQ 机器人（WebSocket 模式）")

	c.done = make(chan struct{})
	c.stopOnce = sync.Once{}

	credentials := &token.QQBotCredentials{
		AppID:     c.config.AppID,
		AppSecret: c.config.AppSecret,
	}
	c.tokenSource = token.NewQQBotTokenSource(credentials)

	c.ctx, c.cancel = context.WithCancel(ctx)

	if err := token.StartRefreshAccessToken(c.ctx, c.tokenSource); err != nil {
		if strings.Contains(err.Error(), "ParseInt") || strings.Contains(err.Error(), "parsing") {
			return fmt.Errorf("QQ 机器人凭证无效，请检查 app_id 和 app_secret 配置: %w", err)
		}
		return fmt.Errorf("failed to start token refresh: %w", err)
	}

	c.api = botgo.NewOpenAPI(c.config.AppID, c.tokenSource).WithTimeout(5 * time.Second)

	intent := event.RegisterHandlers(
		c.handleC2CMessage(),
		c.handleGroupATMessage(),
	)

	c.logger.Infof("注册的 Intent 值: %d (0x%x)", intent, intent)
	c.logger.Info("注册的消息处理器: C2CMessage, GroupATMessage")

	wsInfo, err := c.api.WS(c.ctx, nil, "")
	if err != nil {
		return fmt.Errorf("failed to get websocket info: %w", err)
	}

	c.logger.Infof("获取到 WebSocket 信息, shards: %d", wsInfo.Shards)

	c.sessionManager = botgo.NewSessionManager()

	go func() {
		if err := c.sessionManager.Start(wsInfo, c.tokenSource, &intent); err != nil {
			c.logger.Error("WebSocket 会话错误", "error", err)
			c.running.Store(false)
		}
	}()

	go c.dedupJanitor()

	if c.config.ReasoningChannelID != "" {
		c.chatType.Store(c.config.ReasoningChannelID, "group")
	}

	c.running.Store(true)
	c.logger.Info("QQ 机器人启动成功")

	return nil
}

func (c *Channel) Stop(ctx context.Context) error {
	c.logger.Info("停止 QQ 机器人")
	c.running.Store(false)

	c.stopOnce.Do(func() { close(c.done) })

	if c.cancel != nil {
		c.cancel()
	}

	return nil
}

// Send 发送消息到 QQ 机器人
// msg 要发送的消息
// 返回错误信息
func (c *Channel) Send(ctx context.Context, msg models.OutboundMessage) error {
	if !c.running.Load() {
		return errs.ErrNotRunning
	}

	chatKind := c.getChatKind(msg.SessionID)

	msgToCreate := &dto.MessageToCreate{
		Content: msg.Text,
		MsgType: dto.TextMsg,
	}

	if c.config.SendMarkdown {
		msgToCreate.MsgType = dto.MarkdownMsg
		msgToCreate.Markdown = &dto.Markdown{
			Content: msg.Text,
		}
		msgToCreate.Content = ""
	}

	if v, ok := c.lastMsgID.Load(msg.SessionID); ok {
		if msgID, ok := v.(string); ok && msgID != "" {
			msgToCreate.MsgID = msgID

			if counterVal, ok := c.msgSeqCounters.Load(msg.SessionID); ok {
				if counter, ok := counterVal.(*atomic.Uint64); ok {
					seq := counter.Add(1)
					msgToCreate.MsgSeq = uint32(seq)
				}
			}
		}
	}

	if chatKind == "group" {
		if msgToCreate.Content != "" {
			msgToCreate.Content = sanitizeURLs(msgToCreate.Content)
		}
		if msgToCreate.Markdown != nil && msgToCreate.Markdown.Content != "" {
			msgToCreate.Markdown.Content = sanitizeURLs(msgToCreate.Markdown.Content)
		}
	}

	var err error
	if chatKind == "group" {
		_, err = c.api.PostGroupMessage(ctx, msg.SessionID, msgToCreate)
	} else {
		_, err = c.api.PostC2CMessage(ctx, msg.SessionID, msgToCreate)
	}

	if err != nil {
		c.logger.Error("发送消息失败", "error", err)
		return fmt.Errorf("qq send: %w", errs.ErrTemporary)
	}

	return nil
}

func (c *Channel) IsRunning() bool {
	return c.running.Load()
}

func (c *Channel) IsAllowed(senderID string) bool {
	if len(c.config.AllowFrom) == 0 {
		return true
	}
	for _, allowed := range c.config.AllowFrom {
		if allowed == senderID {
			return true
		}
	}
	return false
}

func (c *Channel) IsAllowedSender(sender models.SenderInfo) bool {
	return c.IsAllowed(sender.ID)
}

func (c *Channel) ReasoningChannelID() string {
	return c.config.ReasoningChannelID
}

func (c *Channel) getChatKind(chatID string) string {
	if v, ok := c.chatType.Load(chatID); ok {
		if k, ok := v.(string); ok {
			return k
		}
	}
	c.logger.Debugf("未知的聊天类型 for chatID: %s, 默认为 group", chatID)
	return "group"
}

func (c *Channel) handleC2CMessage() event.C2CMessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSC2CMessageData) error {
		if c.isDuplicate(data.ID) {
			return nil
		}

		var senderID string
		if data.Author != nil && data.Author.ID != "" {
			senderID = data.Author.ID
		} else {
			c.logger.Warn("收到无发送者 ID 的消息")
			return nil
		}

		content := data.Content
		if content == "" {
			c.logger.Debug("收到空消息，忽略")
			return nil
		}

		c.chatType.Store(senderID, "direct")
		c.lastMsgID.Store(senderID, data.ID)
		c.msgSeqCounters.Store(senderID, new(atomic.Uint64))

		sender := bus.SenderInfo{
			ID:   data.Author.ID,
			Name: data.Author.Username,
		}

		c.logger.Infof("收到 C2C 消息, sender: %s, group: %s, length: %d", senderID, data.GroupID, len(content))

		inboundMsg := bus.InboundMessage{
			Channel:   c.name,
			SessionID: senderID,
			Sender:    sender,
			Text:      content,
			Timestamp: time.Now(),
			Metadata:  map[string]any{"account_id": senderID},
		}

		if err := c.bus.PublishInbound(c.ctx, inboundMsg); err != nil {
			c.logger.Errorf("发布消息到总线失败: %v", err)
		} else {
			c.logger.Infof(">>> QQ发布消息到总线成功, channel=%s, session_id=%s, text=%s", inboundMsg.Channel, inboundMsg.SessionID, inboundMsg.Text)
		}

		return nil
	}
}

func (c *Channel) handleGroupATMessage() event.GroupATMessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSGroupATMessageData) error {
		if c.isDuplicate(data.ID) {
			return nil
		}

		var senderID string
		if data.Author != nil && data.Author.ID != "" {
			senderID = data.Author.ID
		} else {
			c.logger.Warn("收到无发送者 ID 的群消息")
			return nil
		}

		content := data.Content
		if content == "" {
			c.logger.Debug("收到空群消息，忽略")
			return nil
		}

		respond, cleaned := shouldRespondInGroup(content, c.config.GroupTrigger)
		if !respond {
			return nil
		}
		content = cleaned

		c.logger.Infof("收到群 @ 消息, sender: %s, group: %s, length: %d", senderID, data.GroupID, len(content))

		c.chatType.Store(data.GroupID, "group")
		c.lastMsgID.Store(data.GroupID, data.ID)
		c.msgSeqCounters.Store(data.GroupID, new(atomic.Uint64))

		sender := bus.SenderInfo{
			ID:   data.Author.ID,
			Name: data.Author.Username,
		}

		inboundMsg := bus.InboundMessage{
			Channel:   c.name,
			SessionID: data.GroupID,
			Sender:    sender,
			Text:      content,
			Timestamp: time.Now(),
			Metadata:  map[string]any{"account_id": senderID, "group_id": data.GroupID},
		}

		c.bus.PublishInbound(c.ctx, inboundMsg)

		return nil
	}
}

func shouldRespondInGroup(content, groupTrigger string) (bool, string) {
	if groupTrigger == "" {
		return true, content
	}

	atBotPattern := regexp.MustCompile(`@\s*Bot\s*`)
	matches := atBotPattern.FindStringIndex(content)
	if matches == nil {
		return false, content
	}

	cleaned := atBotPattern.ReplaceAllString(content, "")
	cleaned = strings.TrimSpace(cleaned)
	return true, cleaned
}

func (c *Channel) isDuplicate(messageID string) bool {
	c.muDedup.Lock()
	defer c.muDedup.Unlock()

	if ts, exists := c.dedup[messageID]; exists && time.Since(ts) < dedupTTL {
		return true
	}

	if len(c.dedup) >= dedupMaxSize {
		var oldestID string
		var oldestTS time.Time
		for id, ts := range c.dedup {
			if oldestID == "" || ts.Before(oldestTS) {
				oldestID = id
				oldestTS = ts
			}
		}
		if oldestID != "" {
			delete(c.dedup, oldestID)
		}
	}

	c.dedup[messageID] = time.Now()
	return false
}

func (c *Channel) dedupJanitor() {
	ticker := time.NewTicker(dedupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.muDedup.Lock()
			now := time.Now()
			var expired []string
			for id, ts := range c.dedup {
				if now.Sub(ts) >= dedupTTL {
					expired = append(expired, id)
				}
			}
			for _, id := range expired {
				delete(c.dedup, id)
			}
			var mediaExpired []string
			for id, ts := range c.mediaDedup {
				if now.Sub(ts) >= mediaDedupTTL {
					mediaExpired = append(mediaExpired, id)
				}
			}
			for _, id := range mediaExpired {
				delete(c.mediaDedup, id)
			}
			c.muDedup.Unlock()
		}
	}
}

func isHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

var urlPattern = regexp.MustCompile(
	`(?i)` +
		`https?://` +
		`(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+` +
		`[a-zA-Z]{2,}` +
		`(?:[/?#]\S*)?`,
)

func sanitizeURLs(text string) string {
	return urlPattern.ReplaceAllStringFunc(text, func(match string) string {
		idx := strings.Index(match, "://")
		scheme := match[:idx+3]
		rest := match[idx+3:]

		domainEnd := len(rest)
		for i, ch := range rest {
			if ch == '/' || ch == '?' || ch == '#' {
				domainEnd = i
				break
			}
		}

		domain := rest[:domainEnd]
		path := rest[domainEnd:]

		domain = strings.ReplaceAll(domain, ".", "。")

		return scheme + domain + path
	})
}

func (c *Channel) isMediaDuplicate(mediaKey string) bool {
	c.muDedup.Lock()
	defer c.muDedup.Unlock()

	if ts, exists := c.mediaDedup[mediaKey]; exists && time.Since(ts) < mediaDedupTTL {
		return true
	}

	if len(c.mediaDedup) >= mediaDedupMaxSize {
		var oldestID string
		var oldestTS time.Time
		for id, ts := range c.mediaDedup {
			if oldestID == "" || ts.Before(oldestTS) {
				oldestID = id
				oldestTS = ts
			}
		}
		if oldestID != "" {
			delete(c.mediaDedup, oldestID)
		}
	}

	c.mediaDedup[mediaKey] = time.Now()
	return false
}

func (c *Channel) SendImage(ctx context.Context, sessionID string, imageURL string, isGroup bool) error {
	if !c.running.Load() {
		return errs.ErrNotRunning
	}

	c.logger.Infof("发送图片消息, session: %s, isGroup: %v", sessionID, isGroup)

	msgToCreate := &dto.MessageToCreate{
		Content: imageURL,
	}

	var err error
	if isGroup {
		_, err = c.api.PostGroupMessage(ctx, sessionID, msgToCreate)
	} else {
		_, err = c.api.PostC2CMessage(ctx, sessionID, msgToCreate)
	}

	if err != nil {
		c.logger.Error("发送图片消息失败", "error", err)
		return fmt.Errorf("qq send image: %w", errs.ErrTemporary)
	}

	return nil
}

func (c *Channel) SendFile(ctx context.Context, sessionID string, fileURL string, fileName string, isGroup bool) error {
	if !c.running.Load() {
		return errs.ErrNotRunning
	}

	c.logger.Infof("发送文件消息, session: %s, file: %s, isGroup: %v", sessionID, fileName, isGroup)

	msgToCreate := &dto.MessageToCreate{
		Content: fileURL,
	}

	var err error
	if isGroup {
		_, err = c.api.PostGroupMessage(ctx, sessionID, msgToCreate)
	} else {
		_, err = c.api.PostC2CMessage(ctx, sessionID, msgToCreate)
	}

	if err != nil {
		c.logger.Error("发送文件消息失败", "error", err)
		return fmt.Errorf("qq send file: %w", errs.ErrTemporary)
	}

	return nil
}

func (c *Channel) handleIncomingImage(sessionID string, sender bus.SenderInfo, imageKey string, metadata map[string]any) {
	mediaKey := fmt.Sprintf("%s-%s", sessionID, imageKey)
	if c.isMediaDuplicate(mediaKey) {
		c.logger.Debugf("收到重复媒体消息, key: %s, 忽略", mediaKey)
		return
	}

	inboundMsg := bus.InboundMessage{
		Channel:   c.name,
		SessionID: sessionID,
		Sender:    sender,
		Text:      "",
		Media:     []string{imageKey},
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	if err := c.bus.PublishInbound(c.ctx, inboundMsg); err != nil {
		c.logger.Errorf("发布媒体消息到总线失败: %v", err)
	} else {
		c.logger.Infof(">>> QQ发布媒体消息到总线成功, channel=%s, session_id=%s, media=%s", inboundMsg.Channel, inboundMsg.SessionID, imageKey)
	}
}
