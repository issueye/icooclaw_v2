package channels

import (
	"context"
	"errors"
	"icooclaw/pkg/adapter"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels/dingtalk"
	"icooclaw/pkg/channels/feishu"
	"icooclaw/pkg/channels/icoo_chat"
	"icooclaw/pkg/channels/models"
	"icooclaw/pkg/channels/qq"
	"icooclaw/pkg/channels/websocket"
	"icooclaw/pkg/consts"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
)

// Manager 渠道管理器，负责管理多个渠道的通信
type Manager struct {
	// logger 日志记录器
	logger *slog.Logger
	// bus 消息总线
	bus *bus.MessageBus
	// storage 存储渠道配置的存储器
	storage adapter.StorageInterface
	// channels 渠道注册表
	channels map[string]models.Channel
	// lock 锁，用于保护 channels 字段
	lock sync.RWMutex
	// running 运行状态
	running atomic.Bool
	// listenStarted Listen goroutine 启动完成信号
	listenStarted chan struct{}
	// router 路由器，用于处理 HTTP 请求
	router chi.Router
}

// NewManager 创建一个新的渠道管理器
func NewManager(logger *slog.Logger, r chi.Router, bus *bus.MessageBus, storage adapter.StorageInterface) *Manager {
	return &Manager{
		logger:        logger,
		bus:           bus,
		storage:       storage,
		channels:      make(map[string]models.Channel),
		lock:          sync.RWMutex{},
		listenStarted: make(chan struct{}),
		router:        r,
	}
}

// Start 启动渠道管理器
func (m *Manager) Start(ctx context.Context) error {
	if m.storage == nil {
		return errors.New("storage is nil")
	}

	// 启动所有渠道
	err := m.startChannel(ctx)
	if err != nil {
		m.logger.Error("启动管道失败", slog.Any("err", err.Error()))
	}

	// 启动监听（goroutine 中运行，不阻塞）
	m.Listen(ctx)
	// 等待 Listen goroutine 完成初始化，避免竞态条件
	select {
	case <-m.listenStarted:
		m.logger.Info("渠道管理器启动完成，Listen 已就绪")
	case <-ctx.Done():
		return ctx.Err()
	}
	// 标记为运行中
	m.running.Store(true)
	return nil
}

// Listen bus 监听消息总线上的事件（outbound 消息路由到对应渠道发送）
func (m *Manager) Listen(ctx context.Context) {
	go func() {
		m.logger.Info("outbound监听已启动")
		// 通知 Start 方法，Listen goroutine 已启动
		close(m.listenStarted)
		for {
			select {
			case <-ctx.Done():
				m.logger.Info("outbound监听已停止", "reason", ctx.Err())
				return
			case msg, ok := <-m.bus.Outbound():
				if !ok {
					m.logger.Warn("outbound监听：总线通道已关闭")
					return
				}
				m.logger.Info("outbound路由：收到消息", "channel", msg.Channel, "session_id", msg.SessionID)
				m.routeOutbound(ctx, msg)
			}
		}
	}()
}

// routeOutbound 将 outbound 消息路由到对应渠道
func (m *Manager) routeOutbound(ctx context.Context, msg bus.OutboundMessage) {
	m.lock.RLock()
	ch, ok := m.channels[msg.Channel]
	m.lock.RUnlock()

	// 检查渠道是否存在
	if !ok {
		m.logger.Warn("outbound路由：未找到渠道", "channel", msg.Channel, "session_id", msg.SessionID)
		return
	}

	// 发送消息到渠道
	err := ch.Send(ctx, models.BusToOutMessage(msg))
	if err != nil {
		m.logger.Error("outbound路由：发送失败", "channel", msg.Channel, "session_id", msg.SessionID, "error", err)
	}
}

// Remove 移除指定类型的渠道
func (m *Manager) Remove(ctx context.Context, channelType string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	// 停止渠道
	err := m.channels[channelType].Stop(ctx)
	if err != nil {
		m.logger.Error("停止渠道失败", "channel", channelType, "error", err)
	}

	delete(m.channels, channelType)
}

// Add 添加指定类型的渠道实例
func (m *Manager) Add(ctx context.Context, channel string, cfg string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	// 创建渠道实例
	switch channel {
	case consts.QQ: // 创建 QQ 渠道实例
		ch, err := qq.New(m.logger, m.bus, cfg)
		if err != nil {
			return err
		}
		m.channels[consts.QQ] = ch
		// 启动渠道
		if err := ch.Start(ctx); err != nil {
			return err
		}
	case consts.FEISHU_CN, consts.FEISHU: // 创建飞书渠道实例
		ch, err := feishu.New(m.logger, m.bus, cfg)
		if err != nil {
			return err
		}
		m.channels[consts.FEISHU_CN] = ch
		// 启动渠道
		if err := ch.Start(ctx); err != nil {
			return err
		}
	case consts.DINGTALK: // 创建钉钉渠道实例
		ch, err := dingtalk.New(m.logger, m.bus, cfg)
		if err != nil {
			return err
		}
		m.channels[consts.DINGTALK] = ch
		// 启动渠道
		if err := ch.Start(ctx); err != nil {
			return err
		}
	case consts.ICOO_CHAT: // 创建 icooclaw 渠道实例
		ch, err := icoo_chat.New(m.logger, m.bus, cfg)
		if err != nil {
			return err
		}
		m.channels[consts.ICOO_CHAT] = ch
		if err := ch.Start(ctx); err != nil {
			return err
		}
	case consts.WEBSOCKET:
		wsCfg := websocket.DefaultManagerConfig()
		ch := websocket.NewManager(wsCfg, m.logger, m.router, m.bus)
		m.channels[consts.WEBSOCKET] = ch
		if err := ch.Start(ctx); err != nil {
			return err
		}
	default:
		m.logger.Warn("未知渠道类型，已跳过", "type", channel)
	}

	return nil
}

// GetChannel 根据渠道类型获取渠道实例
func (m *Manager) GetChannel(channelType string) models.Channel {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.channels[channelType]
}

// startChannel 启动所有渠道
func (m *Manager) startChannel(ctx context.Context) error {
	// 从存储器加载渠道配置
	list, err := m.storage.List()
	if err != nil {
		return err
	}

	// 启动所有渠道
	for _, channel := range list {
		// 创建渠道实例
		err := m.Add(ctx, channel.Type, channel.Config)
		if err != nil {
			return err
		}
	}

	return nil
}

// Stop 停止渠道管理器
func (m *Manager) Stop(ctx context.Context) error {
	var errs []error
	for _, channel := range m.channels {
		if err := channel.Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
