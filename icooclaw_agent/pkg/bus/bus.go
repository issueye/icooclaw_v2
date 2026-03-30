// Package bus 提供组件间通信所需的消息总线。
package bus

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"icooclaw/pkg/errors"
)

// SenderInfo 描述消息发送者信息。
type SenderInfo struct {
	ID       string
	Name     string
	Username string
	IsBot    bool
}

// InboundMessage 表示从渠道接收到的入站消息。
type InboundMessage struct {
	Channel   string
	SessionID string
	Sender    SenderInfo
	Text      string
	Media     []string
	ReplyTo   string
	Timestamp time.Time
	Metadata  map[string]any
}

// OutboundMessage 表示准备发送到渠道的出站消息。
type OutboundMessage struct {
	Channel   string
	SessionID string
	Text      string
	Media     []string
	ReplyTo   string
	EditID    string
	Metadata  map[string]any
}

// OutboundMediaMessage 表示准备发送的媒体消息。
type OutboundMediaMessage struct {
	Channel   string
	SessionID string
	Media     []string
	Caption   string
	Metadata  map[string]any
}

const defaultBusBufferSize = 64

const (
	highWaterMarkPercent = 80
	scaleUpFactor        = 2
	maxBufferSize        = 1024
	metricsInterval      = 30 * time.Second
)

type Metrics struct {
	InboundQueueSize    int   `json:"inbound_queue_size"`
	InboundCapacity     int   `json:"inbound_capacity"`
	InboundUtilization  int   `json:"inbound_utilization_percent"`
	OutboundQueueSize   int   `json:"outbound_queue_size"`
	OutboundCapacity    int   `json:"outbound_capacity"`
	OutboundUtilization int   `json:"outbound_utilization_percent"`
	DropCount           int64 `json:"drop_count"`
	SubscriberCount     int   `json:"subscriber_count"`
	IsClosed            bool  `json:"is_closed"`
}

type AlertType string

const (
	AlertHighUtilization AlertType = "high_utilization"
	AlertDropDetected    AlertType = "drop_detected"
)

type Alert struct {
	Type      AlertType `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Metrics   Metrics   `json:"metrics"`
}

type AlertHandler func(Alert)

type MessageBus struct {
	inbound       chan InboundMessage
	outbound      chan OutboundMessage
	outboundMedia chan OutboundMediaMessage
	done          chan struct{}
	closed        atomic.Bool

	inboundCapacity  int
	outboundCapacity int
	dropCount        atomic.Int64

	inboundSubs  map[string]chan InboundMessage
	outboundSubs map[string]chan OutboundMessage
	mu           sync.RWMutex

	logger       *slog.Logger
	alertHandler AlertHandler
	stopMetrics  chan struct{}
}

// Config 保存 MessageBus 配置。
type Config struct {
	InboundCapacity  int
	OutboundCapacity int
}

// DefaultConfig 返回默认配置。
func DefaultConfig() Config {
	return Config{
		InboundCapacity:  defaultBusBufferSize,
		OutboundCapacity: defaultBusBufferSize,
	}
}

// NewMessageBus 创建一个新的 MessageBus。
func NewMessageBus(cfg Config) *MessageBus {
	if cfg.InboundCapacity <= 0 {
		cfg.InboundCapacity = defaultBusBufferSize
	}
	if cfg.OutboundCapacity <= 0 {
		cfg.OutboundCapacity = defaultBusBufferSize
	}

	return &MessageBus{
		inbound:          make(chan InboundMessage, cfg.InboundCapacity),
		outbound:         make(chan OutboundMessage, cfg.OutboundCapacity),
		outboundMedia:    make(chan OutboundMediaMessage, cfg.InboundCapacity),
		done:             make(chan struct{}),
		inboundCapacity:  cfg.InboundCapacity,
		outboundCapacity: cfg.OutboundCapacity,
		inboundSubs:      make(map[string]chan InboundMessage),
		outboundSubs:     make(map[string]chan OutboundMessage),
		logger:           slog.Default(),
		stopMetrics:      make(chan struct{}),
	}
}

// PublishInbound 在支持上下文控制的情况下发布入站消息。
// 如果总线已关闭则返回 ErrBusClosed，如果上下文取消则返回 ctx.Err()。
func (mb *MessageBus) PublishInbound(ctx context.Context, msg InboundMessage) error {
	if mb.closed.Load() {
		return errors.ErrNotRunning
	}

	// 发送前先检查上下文状态
	if err := ctx.Err(); err != nil {
		return err
	}

	select {
	case mb.inbound <- msg:
		// 同时转发给订阅者，并对已关闭通道做 panic 保护
		mb.forwardToInboundSubscribers(msg)
		return nil
	case <-mb.done:
		return errors.ErrNotRunning
	case <-ctx.Done():
		return ctx.Err()
	}
}

// forwardToInboundSubscribers 将消息转发给入站订阅者，并处理已关闭通道的 panic。
func (mb *MessageBus) forwardToInboundSubscribers(msg InboundMessage) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	for name, sub := range mb.inboundSubs {
		// 使用 recover 处理向已关闭通道发送消息的情况
		func() {
			defer func() {
				if r := recover(); r != nil {
					// 通道已关闭，订阅者可能已经被移除
					mb.dropCount.Add(1)
				}
			}()
			select {
			case sub <- msg:
			default:
				// 订阅者缓冲区已满，记为丢弃
				mb.dropCount.Add(1)
				_ = name // 避免未使用变量告警
			}
		}()
	}
}

// PublishInboundNoCtx 在不传入上下文的情况下发布入站消息（仅保留兼容性）。
// Deprecated: 请改用带上下文的 PublishInbound。
func (mb *MessageBus) PublishInboundNoCtx(msg InboundMessage) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return mb.PublishInbound(ctx, msg)
}

// ConsumeInbound 从总线消费一条入站消息。
// 成功时返回消息和 true；总线关闭或上下文取消时返回空消息和 false。
func (mb *MessageBus) ConsumeInbound(ctx context.Context) (InboundMessage, bool) {
	select {
	case msg, ok := <-mb.inbound:
		return msg, ok
	case <-mb.done:
		return InboundMessage{}, false
	case <-ctx.Done():
		return InboundMessage{}, false
	}
}

// PublishOutbound 在支持上下文控制的情况下发布出站消息。
// 如果总线已关闭则返回 ErrBusClosed，如果上下文取消则返回 ctx.Err()。
func (mb *MessageBus) PublishOutbound(ctx context.Context, msg OutboundMessage) error {
	if mb.closed.Load() {
		return errors.ErrNotRunning
	}

	// 发送前先检查上下文状态
	if err := ctx.Err(); err != nil {
		return err
	}

	select {
	case mb.outbound <- msg:
		// 同时转发给订阅者，并对已关闭通道做 panic 保护
		mb.forwardToOutboundSubscribers(msg)
		return nil
	case <-mb.done:
		return errors.ErrNotRunning
	case <-ctx.Done():
		return ctx.Err()
	}
}

// forwardToOutboundSubscribers 将消息转发给出站订阅者，并处理已关闭通道的 panic。
func (mb *MessageBus) forwardToOutboundSubscribers(msg OutboundMessage) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	for name, sub := range mb.outboundSubs {
		// 使用 recover 处理向已关闭通道发送消息的情况
		func() {
			defer func() {
				if r := recover(); r != nil {
					// 通道已关闭，订阅者可能已经被移除
					mb.dropCount.Add(1)
				}
			}()
			select {
			case sub <- msg:
			default:
				// 订阅者缓冲区已满，记为丢弃
				mb.dropCount.Add(1)
				_ = name // 避免未使用变量告警
			}
		}()
	}
}

// PublishOutboundNoCtx 在不传入上下文的情况下发布出站消息（仅保留兼容性）。
// Deprecated: 请改用带上下文的 PublishOutbound。
func (mb *MessageBus) PublishOutboundNoCtx(msg OutboundMessage) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return mb.PublishOutbound(ctx, msg)
}

// ConsumeOutbound 从总线消费一条出站消息。
// 成功时返回消息和 true；总线关闭或上下文取消时返回空消息和 false。
func (mb *MessageBus) ConsumeOutbound(ctx context.Context) (OutboundMessage, bool) {
	select {
	case msg, ok := <-mb.outbound:
		return msg, ok
	case <-mb.done:
		return OutboundMessage{}, false
	case <-ctx.Done():
		return OutboundMessage{}, false
	}
}

// PublishOutboundMedia 在支持上下文控制的情况下发布出站媒体消息。
func (mb *MessageBus) PublishOutboundMedia(ctx context.Context, msg OutboundMediaMessage) error {
	if mb.closed.Load() {
		return errors.ErrNotRunning
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	select {
	case mb.outboundMedia <- msg:
		return nil
	case <-mb.done:
		return errors.ErrNotRunning
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ConsumeOutboundMedia 从总线消费一条出站媒体消息。
func (mb *MessageBus) ConsumeOutboundMedia(ctx context.Context) (OutboundMediaMessage, bool) {
	select {
	case msg, ok := <-mb.outboundMedia:
		return msg, ok
	case <-mb.done:
		return OutboundMediaMessage{}, false
	case <-ctx.Done():
		return OutboundMediaMessage{}, false
	}
}

// Inbound 返回入站消息通道。
// Deprecated: 请改用带上下文的 ConsumeInbound，以获得更安全的消费方式。
func (mb *MessageBus) Inbound() <-chan InboundMessage {
	return mb.inbound
}

// Outbound 返回出站消息通道。
// Deprecated: 请改用带上下文的 ConsumeOutbound，以获得更安全的消费方式。
func (mb *MessageBus) Outbound() <-chan OutboundMessage {
	return mb.outbound
}

// OutboundMedia 返回出站媒体消息通道。
// Deprecated: 请改用带上下文的 ConsumeOutboundMedia，以获得更安全的消费方式。
func (mb *MessageBus) OutboundMedia() <-chan OutboundMediaMessage {
	return mb.outboundMedia
}

// SubscribeInbound 订阅入站消息。
// 如果存在同名订阅，会替换旧订阅并关闭旧通道。
func (mb *MessageBus) SubscribeInbound(name string, buffer int) <-chan InboundMessage {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if buffer <= 0 {
		buffer = 100
	}

	// 如已存在同名订阅则先关闭，避免通道泄漏
	if oldCh, ok := mb.inboundSubs[name]; ok {
		close(oldCh)
	}

	ch := make(chan InboundMessage, buffer)
	mb.inboundSubs[name] = ch
	return ch
}

// SubscribeOutbound 订阅出站消息。
// 如果存在同名订阅，会替换旧订阅并关闭旧通道。
func (mb *MessageBus) SubscribeOutbound(name string, buffer int) <-chan OutboundMessage {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if buffer <= 0 {
		buffer = 100
	}

	// 如已存在同名订阅则先关闭，避免通道泄漏
	if oldCh, ok := mb.outboundSubs[name]; ok {
		close(oldCh)
	}

	ch := make(chan OutboundMessage, buffer)
	mb.outboundSubs[name] = ch
	return ch
}

// UnsubscribeInbound 取消入站消息订阅。
func (mb *MessageBus) UnsubscribeInbound(name string) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if ch, ok := mb.inboundSubs[name]; ok {
		close(ch)
		delete(mb.inboundSubs, name)
	}
}

// UnsubscribeOutbound 取消出站消息订阅。
func (mb *MessageBus) UnsubscribeOutbound(name string) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if ch, ok := mb.outboundSubs[name]; ok {
		close(ch)
		delete(mb.outboundSubs, name)
	}
}

// Close 优雅关闭消息总线。
// 它会尽量清空缓冲区，避免并发发布时出现向已关闭通道发送消息的 panic。
// 这里不会关闭主通道，以避免并发发布场景下的 send-on-closed panic。
func (mb *MessageBus) Close() {
	if mb.closed.CompareAndSwap(false, true) {
		close(mb.done)

		// 尽量排空缓冲区，避免消息被静默丢弃。
		// 这里不会关闭主通道，以避免并发发布场景下的 send-on-closed panic。
		drained := 0
		for {
			select {
			case <-mb.inbound:
				drained++
			default:
				goto doneInbound
			}
		}
	doneInbound:
		for {
			select {
			case <-mb.outbound:
				drained++
			default:
				goto doneOutbound
			}
		}
	doneOutbound:
		for {
			select {
			case <-mb.outboundMedia:
				drained++
			default:
				goto doneMedia
			}
		}
	doneMedia:
		_ = drained // 避免未使用变量告警

		// 关闭订阅者通道
		mb.mu.Lock()
		for _, ch := range mb.inboundSubs {
			close(ch)
		}
		for _, ch := range mb.outboundSubs {
			close(ch)
		}
		mb.inboundSubs = make(map[string]chan InboundMessage)
		mb.outboundSubs = make(map[string]chan OutboundMessage)
		mb.mu.Unlock()
	}
}

// Done 返回 done 通道。
func (mb *MessageBus) Done() <-chan struct{} {
	return mb.done
}

// IsClosed 返回总线是否已关闭。
func (mb *MessageBus) IsClosed() bool {
	return mb.closed.Load()
}

// DropCount 返回已丢弃消息数量。
func (mb *MessageBus) DropCount() int64 {
	return mb.dropCount.Load()
}

// Run 启动消息总线（为兼容保留；当前实现中通道创建后即处于可用状态）。
func (mb *MessageBus) Run(ctx context.Context) error {
	<-ctx.Done()
	mb.Close()
	return ctx.Err()
}

// GetMetrics 返回当前指标。
func (mb *MessageBus) GetMetrics() Metrics {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	inboundLen := len(mb.inbound)
	outboundLen := len(mb.outbound)

	inboundUtil := 0
	if mb.inboundCapacity > 0 {
		inboundUtil = (inboundLen * 100) / mb.inboundCapacity
	}
	outboundUtil := 0
	if mb.outboundCapacity > 0 {
		outboundUtil = (outboundLen * 100) / mb.outboundCapacity
	}

	return Metrics{
		InboundQueueSize:    inboundLen,
		InboundCapacity:     mb.inboundCapacity,
		InboundUtilization:  inboundUtil,
		OutboundQueueSize:   outboundLen,
		OutboundCapacity:    mb.outboundCapacity,
		OutboundUtilization: outboundUtil,
		DropCount:           mb.dropCount.Load(),
		SubscriberCount:     len(mb.inboundSubs) + len(mb.outboundSubs),
		IsClosed:            mb.closed.Load(),
	}
}

// SetLogger 设置日志记录器。
func (mb *MessageBus) SetLogger(logger *slog.Logger) {
	mb.logger = logger
}

// SetAlertHandler 设置告警处理函数。
func (mb *MessageBus) SetAlertHandler(handler AlertHandler) {
	mb.alertHandler = handler
}

// StartMetricsReporter 启动指标上报线程。
func (mb *MessageBus) StartMetricsReporter(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(metricsInterval)
		defer ticker.Stop()

		var lastDropCount int64
		for {
			select {
			case <-ctx.Done():
				return
			case <-mb.stopMetrics:
				return
			case <-ticker.C:
				metrics := mb.GetMetrics()

				if mb.alertHandler != nil {
					currentDropCount := metrics.DropCount
					if currentDropCount > lastDropCount {
						alert := Alert{
							Type:      AlertDropDetected,
							Message:   fmt.Sprintf("Message drop detected: %d drops (total: %d)", currentDropCount-lastDropCount, currentDropCount),
							Timestamp: time.Now(),
							Metrics:   metrics,
						}
						mb.alertHandler(alert)
					}
					lastDropCount = currentDropCount

					if metrics.InboundUtilization >= highWaterMarkPercent || metrics.OutboundUtilization >= highWaterMarkPercent {
						alert := Alert{
							Type:      AlertHighUtilization,
							Message:   fmt.Sprintf("High queue utilization: inbound=%d%%, outbound=%d%%", metrics.InboundUtilization, metrics.OutboundUtilization),
							Timestamp: time.Now(),
							Metrics:   metrics,
						}
						mb.alertHandler(alert)
					}
				}

				if mb.logger != nil {
					mb.logger.Debug("总线指标",
						"inbound_util", metrics.InboundUtilization,
						"outbound_util", metrics.OutboundUtilization,
						"drop_count", metrics.DropCount,
					)
				}
			}
		}
	}()
}

// StopMetricsReporter 停止指标上报线程。
func (mb *MessageBus) StopMetricsReporter() {
	select {
	case mb.stopMetrics <- struct{}{}:
	default:
	}
}

// checkAndScale 检查并调整缓冲区大小。
func (mb *MessageBus) checkAndScale(ctx context.Context) {
	metrics := mb.GetMetrics()

	if metrics.InboundUtilization >= highWaterMarkPercent && mb.inboundCapacity < maxBufferSize {
		newCapacity := mb.inboundCapacity * scaleUpFactor
		if newCapacity > maxBufferSize {
			newCapacity = maxBufferSize
		}
		mb.scaleInbound(ctx, newCapacity)
	}

	if metrics.OutboundUtilization >= highWaterMarkPercent && mb.outboundCapacity < maxBufferSize {
		newCapacity := mb.outboundCapacity * scaleUpFactor
		if newCapacity > maxBufferSize {
			newCapacity = maxBufferSize
		}
		mb.scaleOutbound(ctx, newCapacity)
	}
}

// scaleInbound 调整入站缓冲区大小。
func (mb *MessageBus) scaleInbound(ctx context.Context, newCapacity int) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if newCapacity <= mb.inboundCapacity {
		return
	}

	oldInbound := mb.inbound
	mb.inbound = make(chan InboundMessage, newCapacity)
	mb.inboundCapacity = newCapacity

	go func() {
		for {
			select {
			case msg := <-oldInbound:
				if mb.closed.Load() {
					return
				}
				select {
				case mb.inbound <- msg:
				default:
					mb.dropCount.Add(1)
				}
			case <-ctx.Done():
				return
			case <-mb.done:
				return
			}
		}
	}()

	if mb.logger != nil {
		mb.logger.Info("Bus inbound scaled",
			"old_capacity", newCapacity/scaleUpFactor,
			"new_capacity", newCapacity,
		)
	}
}

// scaleOutbound 调整出站缓冲区大小。
func (mb *MessageBus) scaleOutbound(ctx context.Context, newCapacity int) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if newCapacity <= mb.outboundCapacity {
		return
	}

	oldOutbound := mb.outbound
	mb.outbound = make(chan OutboundMessage, newCapacity)
	mb.outboundCapacity = newCapacity

	go func() {
		for {
			select {
			case msg := <-oldOutbound:
				if mb.closed.Load() {
					return
				}
				select {
				case mb.outbound <- msg:
				default:
					mb.dropCount.Add(1)
				}
			case <-ctx.Done():
				return
			case <-mb.done:
				return
			}
		}
	}()

	if mb.logger != nil {
		mb.logger.Info("Bus outbound scaled",
			"old_capacity", newCapacity/scaleUpFactor,
			"new_capacity", newCapacity,
		)
	}
}
