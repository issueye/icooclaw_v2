package feishu

import "icooclaw/pkg/channels/models"

// Config 飞书渠道配置结构体.
type Config struct {
	Enabled           bool         `json:"enabled" mapstructure:"enabled"`
	AppID             string       `json:"app_id" mapstructure:"app_id"`
	AppSecret         string       `json:"app_secret" mapstructure:"app_secret"`
	EncryptKey        string       `json:"encrypt_key" mapstructure:"encrypt_key"`
	VerificationToken string       `json:"verification_token" mapstructure:"verification_token"`
	ReasoningChatID   string       `json:"reasoning_chat_id" mapstructure:"reasoning_chat_id"`
	AllowFrom         models.Allow `json:"allow_from" mapstructure:"allow_from"`
}
