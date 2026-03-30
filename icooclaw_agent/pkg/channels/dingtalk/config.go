package dingtalk

import "icooclaw/pkg/channels/models"

// Config 钉钉渠道配置.
type Config struct {
	Enabled         bool         `json:"enabled" mapstructure:"enabled"`
	ClientID        string       `json:"client_id" mapstructure:"client_id"`
	ClientSecret    string       `json:"client_secret" mapstructure:"client_secret"`
	AgentID         int64        `json:"agent_id" mapstructure:"agent_id"`
	ReasoningChatID string       `json:"reasoning_chat_id" mapstructure:"reasoning_chat_id"`
	AllowFrom       models.Allow `json:"allow_from" mapstructure:"allow_from"`
}
