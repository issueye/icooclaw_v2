package agent

import (
	"context"
	"fmt"
	"icooclaw/pkg/agent/react"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	icooclawErrors "icooclaw/pkg/errors"
	"icooclaw/pkg/memory"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/skill"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	outboundEventTypeKey   = "event_type"
	outboundReasoningKey   = "reasoning"
	outboundToolCallIDKey  = "tool_call_id"
	outboundToolNameKey    = "tool_name"
	outboundToolArgsKey    = "tool_args"
	outboundResultKey      = "result"
	outboundIterationKey   = "iteration"
	outboundTotalTokensKey = "total_tokens"
)

const (
	outboundEventChunk      = "chunk"
	outboundEventToolCall   = "tool_call"
	outboundEventToolResult = "tool_result"
	outboundEventEnd        = "end"
	outboundEventError      = "error"
)

type Manager interface {
	// 启动智能体循环
	Start() error
	// 停止智能体循环
	Stop() error
}

// AgentManager 智能体管理器
type AgentManager struct {
	// 上下文
	ctx context.Context
	// 是否正在运行
	running atomic.Bool
	// 消息总线
	bus *bus.MessageBus
	// 内存加载器
	memory memory.Loader
	// 技能加载器
	skills skill.Loader
	// 工具注册器
	tools *tools.Registry
	// 日志记录器
	logger *slog.Logger
	// 钩子函数
	hooks react.ReactHooks
	// 提供商工厂
	providerManager *providers.Manager
	// 存储加载器
	storage *storage.Storage
	// 按会话缓存的智能体实例
	agentsMap map[string]*react.ReActAgent
	agentsMu  sync.RWMutex
	// 运行配置
	maxToolIterations int
}

type preparedAgentRun struct {
	agent *react.ReActAgent
	msg   bus.InboundMessage
}

type Dependencies struct {
	Logger            *slog.Logger
	Bus               *bus.MessageBus
	Memory            memory.Loader
	Skills            skill.Loader
	Tools             *tools.Registry
	Hooks             react.ReactHooks
	ProviderManager   *providers.Manager
	Storage           *storage.Storage
	MaxToolIterations int
}

// NewAgentManager 创建智能体管理器
func NewAgentManager(deps Dependencies) (*AgentManager, error) {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	if deps.Bus == nil {
		return nil, fmt.Errorf("agent manager requires message bus")
	}
	if deps.Tools == nil {
		return nil, fmt.Errorf("agent manager requires tool registry")
	}
	if deps.Skills == nil {
		return nil, fmt.Errorf("agent manager requires skill loader")
	}
	if deps.ProviderManager == nil {
		return nil, fmt.Errorf("agent manager requires provider manager")
	}
	if deps.Storage == nil {
		return nil, fmt.Errorf("agent manager requires storage")
	}
	if deps.MaxToolIterations <= 0 {
		deps.MaxToolIterations = consts.DEFAULT_TOOL_ITERATIONS
	}

	manager := AgentManager{
		running:           atomic.Bool{},
		bus:               deps.Bus,
		memory:            deps.Memory,
		skills:            deps.Skills,
		tools:             deps.Tools,
		logger:            deps.Logger,
		hooks:             deps.Hooks,
		providerManager:   deps.ProviderManager,
		storage:           deps.Storage,
		maxToolIterations: deps.MaxToolIterations,
	}

	manager.agentsMap = make(map[string]*react.ReActAgent)
	return &manager, nil
}

// Start 启动智能体循环
func (m *AgentManager) Start(ctx context.Context) error {
	if m.running.Load() == true {
		return nil
	}

	go m.start(ctx)
	m.running.Store(true)
	return nil
}

func (m *AgentManager) getOrCreateAgent(sessionID string) (*react.ReActAgent, error) {
	m.agentsMu.RLock()
	agent, ok := m.agentsMap[sessionID]
	m.agentsMu.RUnlock()
	if ok {
		return agent, nil
	}

	m.agentsMu.Lock()
	defer m.agentsMu.Unlock()

	if agent, ok = m.agentsMap[sessionID]; ok {
		return agent, nil
	}

	agent, err := react.NewReActAgent(
		m.ctx,
		m.hooks,
		react.Dependencies{
			Tools:             m.tools,
			Memory:            m.memory,
			Skills:            m.skills,
			Storage:           m.storage,
			Bus:               m.bus,
			ProviderManager:   m.providerManager,
			Logger:            m.logger,
			MaxToolIterations: m.maxToolIterations,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("创建智能体失败: %w", err)
	}

	m.agentsMap[sessionID] = agent
	return agent, nil
}

// IsRunning 是否正在运行
func (m *AgentManager) IsRunning() bool {
	return m.running.Load()
}

// start 启动智能体循环
func (m *AgentManager) start(ctx context.Context) error {
	m.ctx = ctx
	// 监听消息总线
	m.logger.Info("代理循环已启动")
	for m.running.Load() {
		select {
		case <-m.ctx.Done():
			m.logger.Info("代理循环已停止", "reason", m.ctx.Err())
			return m.ctx.Err()
		case msg := <-m.bus.Inbound():
			m.logger.Info(">>> Agent收到消息", "channel", msg.Channel, "session_id", msg.SessionID, "sender", msg.Sender.ID, "text", msg.Text)
			switch msg.Channel {
			case consts.WEBSOCKET, consts.ICOO_CHAT:
				// 处理消息
				err := m.RunAgentStream(msg, m.callback(msg))
				if err != nil {
					m.logger.Error("处理消息失败", "reason", err)
					continue
				}
			case consts.FEISHU, consts.FEISHU_CN, consts.QQ:
				m.logger.Debug("收到消息", "channel", msg.Channel, "session_id", msg.SessionID, "text", msg.Text)
				// 处理消息（RunAgent 内部已经发回总线，这里无需再次发送）
				go func(mMsg bus.InboundMessage) {
					_, err := m.RunAgent(mMsg)
					if err != nil {
						m.logger.Error("处理消息失败", "reason", err)
						// 发送错误提示给用户
						errorOut := bus.OutboundMessage{
							Channel:   mMsg.Channel,
							SessionID: mMsg.SessionID,
							Text:      fmt.Sprintf("处理消息时发生错误: %v", err),
						}
						m.bus.PublishOutbound(m.ctx, errorOut)
					}
				}(msg)
			default:
				m.logger.Warn("未知通道类型", "channel", msg.Channel)
			}
		}
	}

	return nil
}

func (m *AgentManager) callback(inbound bus.InboundMessage) react.StreamCallback {
	return func(chunk react.StreamChunk) error {
		for _, out := range buildStreamOutboundMessages(inbound, chunk) {
			if err := m.bus.PublishOutbound(m.ctx, out); err != nil {
				return err
			}
		}
		return nil
	}
}

// Stop 停止智能体循环
func (m *AgentManager) Stop() error {
	if m.running.Load() == false {
		return nil
	}
	m.running.Store(false)
	return nil
}

func (m *AgentManager) prepareAgentRun(msg bus.InboundMessage) (*preparedAgentRun, error) {
	if err := m.ensureSession(msg); err != nil {
		m.logger.Warn("确保会话记录失败", "channel", msg.Channel, "session_id", msg.SessionID, "error", err)
	}

	agent, err := m.getOrCreateAgent(msg.SessionID)
	if err != nil {
		m.logger.Error("创建智能体失败", "reason", err)
		return nil, err
	}

	return &preparedAgentRun{
		agent: agent,
		msg:   msg,
	}, nil
}

func (m *AgentManager) logRunStart(msg bus.InboundMessage, streamed bool) {
	message := "开始处理消息"
	if streamed {
		message = "开始处理流式消息"
	}

	m.logger.Info(message,
		"channel", msg.Channel,
		"session_id", msg.SessionID,
		"sender", msg.Sender.ID,
		"preview", truncate(msg.Text, 50),
		"prompt", msg.Text)
}

func (m *AgentManager) logRunEnd(msg bus.InboundMessage, content string, iteration int, streamed bool) {
	message := "消息处理完成"
	if streamed {
		message = "流式消息处理完成"
	}

	m.logger.Info(message,
		"session_id", msg.SessionID,
		"iteration", iteration,
		"content_len", len(content))
}

func (m *AgentManager) RunAgent(msg bus.InboundMessage) (string, error) {
	prepared, err := m.prepareAgentRun(msg)
	if err != nil {
		return "", err
	}

	m.logRunStart(prepared.msg, false)

	finallyContent, finallyIteration, err := prepared.agent.Chat(m.ctx, prepared.msg)
	if err != nil {
		m.logger.Error("处理消息失败", "reason", err)
		return "", err
	}

	m.logRunEnd(prepared.msg, finallyContent, finallyIteration, false)

	// 将消息发送到消息总线
	out := bus.OutboundMessage{
		Channel:   prepared.msg.Channel,
		SessionID: prepared.msg.SessionID,
		Text:      finallyContent,
		Metadata: map[string]any{
			"iteration": finallyIteration, // 迭代次数
		},
	}
	if err := m.bus.PublishOutbound(m.ctx, out); err != nil {
		m.logger.Error("发送消息到总线失败",
			"session_id", prepared.msg.SessionID,
			"error", err)
	} else {
		m.logger.Debug(">>> 消息已发送到总线",
			"channel", prepared.msg.Channel,
			"session_id", prepared.msg.SessionID,
			"content_len", len(finallyContent))
	}

	// 返回最终内容
	return finallyContent, nil
}

func (m *AgentManager) RunAgentStream(msg bus.InboundMessage, callback react.StreamCallback) error {
	prepared, err := m.prepareAgentRun(msg)
	if err != nil {
		return err
	}

	m.logRunStart(prepared.msg, true)

	finallyContent, finallyIteration, err := prepared.agent.ChatStream(m.ctx, prepared.msg, callback)
	if err != nil {
		m.logger.Error("处理消息失败", "reason", err)
		return err
	}

	m.logRunEnd(prepared.msg, finallyContent, finallyIteration, true)

	return nil
}

func buildStreamOutboundMessages(inbound bus.InboundMessage, chunk react.StreamChunk) []bus.OutboundMessage {
	messages := make([]bus.OutboundMessage, 0, 4)

	appendMessage := func(text string, metadata map[string]any) {
		messages = append(messages, bus.OutboundMessage{
			Channel:   inbound.Channel,
			SessionID: inbound.SessionID,
			Text:      text,
			Metadata:  metadata,
		})
	}

	if chunk.Error != nil {
		appendMessage(chunk.Error.Error(), map[string]any{
			outboundEventTypeKey: outboundEventError,
			outboundIterationKey: chunk.Iteration,
		})
		return messages
	}

	if chunk.ToolName != "" {
		appendMessage("", map[string]any{
			outboundEventTypeKey:  outboundEventToolCall,
			outboundToolCallIDKey: chunk.ToolCallID,
			outboundToolNameKey:   chunk.ToolName,
			outboundToolArgsKey:   chunk.ToolArgs,
			outboundIterationKey:  chunk.Iteration,
		})
	}

	if chunk.ToolResult != "" {
		appendMessage("", map[string]any{
			outboundEventTypeKey:  outboundEventToolResult,
			outboundToolCallIDKey: chunk.ToolCallID,
			outboundToolNameKey:   chunk.ToolName,
			outboundResultKey:     chunk.ToolResult,
			outboundIterationKey:  chunk.Iteration,
		})
	}

	if chunk.Content != "" || chunk.Reasoning != "" {
		metadata := map[string]any{
			outboundEventTypeKey: outboundEventChunk,
			outboundReasoningKey: chunk.Reasoning,
			outboundIterationKey: chunk.Iteration,
		}
		if chunk.TotalTokens > 0 {
			metadata[outboundTotalTokensKey] = chunk.TotalTokens
		}
		appendMessage(chunk.Content, metadata)
	}

	if chunk.Done {
		metadata := map[string]any{
			outboundEventTypeKey: outboundEventEnd,
			outboundIterationKey: chunk.Iteration,
		}
		if chunk.TotalTokens > 0 {
			metadata[outboundTotalTokensKey] = chunk.TotalTokens
		}
		appendMessage("", metadata)
	}

	return messages
}

func (m *AgentManager) ensureSession(msg bus.InboundMessage) error {
	if m.storage == nil || strings.TrimSpace(msg.SessionID) == "" {
		return nil
	}

	session, err := m.storage.Session().GetBySessionID(msg.Channel, msg.SessionID)
	if err != nil && err != icooclawErrors.ErrRecordNotFound {
		return err
	}

	if session == nil {
		session = &storage.Session{
			Model: storage.Model{
				ID: msg.SessionID,
			},
			Channel: msg.Channel,
			UserID:  msg.Sender.ID,
			AgentID: metadataStringValue(msg.Metadata, "agent_id"),
			Title:   conversationTitle(msg.Text),
		}
		return m.storage.Session().Save(session)
	}

	updated := false
	if session.UserID == "" && msg.Sender.ID != "" {
		session.UserID = msg.Sender.ID
		updated = true
	}
	if session.AgentID == "" {
		if agentID := metadataStringValue(msg.Metadata, "agent_id"); agentID != "" {
			session.AgentID = agentID
			updated = true
		}
	}
	if strings.TrimSpace(session.Title) == "" {
		if title := conversationTitle(msg.Text); title != "" {
			session.Title = title
			updated = true
		}
	}

	if updated {
		return m.storage.Session().Save(session)
	}

	return nil
}

func metadataStringValue(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	if value, ok := metadata[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func conversationTitle(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	runes := []rune(content)
	if len(runes) > 30 {
		return string(runes[:30]) + "..."
	}
	return content
}

// truncate 截断字符串到指定长度
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 0 {
		return ""
	}
	return s[:maxLen] + "..."
}
