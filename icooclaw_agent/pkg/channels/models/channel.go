package models

import (
	"context"
)

// Channel 渠道接口，定义了渠道的基本操作
type Channel interface {
	// Start 启动渠道
	Start(ctx context.Context) error
	// Stop 停止渠道
	Stop(ctx context.Context) error
	// Send 发送消息到渠道
	Send(ctx context.Context, msg OutboundMessage) error
	// IsRunning 检查渠道是否正在运行
	IsRunning() bool
	// IsAllowed 检查渠道是否允许指定发送者 ID 发送消息
	IsAllowed(senderID string) bool
}
