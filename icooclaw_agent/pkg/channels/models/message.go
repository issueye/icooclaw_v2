package models

import (
	"icooclaw/pkg/bus"
	"time"
)

// SenderInfo 消息发送者信息
type SenderInfo struct {
	ID       string `json:"id"`       // 发送者 ID
	Name     string `json:"name"`     // 发送者名称
	Username string `json:"username"` // 发送者用户名
	IsBot    bool   `json:"is_bot"`   // 是否为机器人
}

// OutboundMediaMessage 出站媒体消息结构，表示要发送的媒体消息
type OutboundMediaMessage struct {
	Channel   string         `json:"channel"`            // 目标渠道名称
	SessionID string         `json:"session_id"`         // 会话 ID
	Media     []string       `json:"media"`              // 媒体文件 URL 列表
	Caption   string         `json:"caption,omitempty"`  // 媒体说明文字
	Metadata  map[string]any `json:"metadata,omitempty"` // 元数据
}

// OutboundMessage 出站消息结构，表示要发送到渠道的消息
type OutboundMessage struct {
	Channel   string         `json:"channel"`            // 目标渠道名称
	SessionID string         `json:"session_id"`         // 会话 ID
	Text      string         `json:"text"`               // 消息文本内容
	Media     []string       `json:"media,omitempty"`    // 媒体文件 URL 列表
	ReplyTo   string         `json:"reply_to,omitempty"` // 回复的消息 ID
	EditID    string         `json:"edit_id,omitempty"`  // 要编辑的消息 ID
	Metadata  map[string]any `json:"metadata,omitempty"` // 元数据
}

// InboundMessage 入站消息结构，表示从渠道接收到的消息
type InboundMessage struct {
	Channel   string         `json:"channel"`            // 来源渠道名称
	SessionID string         `json:"session_id"`         // 会话 ID
	Sender    SenderInfo     `json:"sender"`             // 发送者信息
	Text      string         `json:"text"`               // 消息文本内容
	Media     []string       `json:"media,omitempty"`    // 媒体文件 URL 列表
	ReplyTo   string         `json:"reply_to,omitempty"` // 回复的消息 ID
	Timestamp time.Time      `json:"timestamp"`          // 消息时间戳
	Metadata  map[string]any `json:"metadata,omitempty"` // 元数据
}

func BusToOutMessage(msg bus.OutboundMessage) OutboundMessage {
	return OutboundMessage{
		Channel:   msg.Channel,
		SessionID: msg.SessionID,
		Text:      msg.Text,
		Media:     msg.Media,
		ReplyTo:   msg.ReplyTo,
		EditID:    msg.EditID,
		Metadata:  msg.Metadata,
	}
}
