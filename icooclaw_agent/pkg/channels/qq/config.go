package qq

import (
	"encoding/json"
	"fmt"
	"icooclaw/pkg/channels/models"
	"strings"
)

type Config struct {
	Enabled            bool         `json:"enabled" mapstructure:"enabled"`
	AppID              string       `json:"app_id" mapstructure:"app_id"`
	AppSecret          string       `json:"app_secret" mapstructure:"app_secret"`
	MaxMessageLength   int          `json:"max_message_length" mapstructure:"max_message_length"`
	GroupTrigger       string       `json:"group_trigger" mapstructure:"group_trigger"`
	ReasoningChannelID string       `json:"reasoning_chat_id" mapstructure:"reasoning_chat_id"`
	SendMarkdown       bool         `json:"send_markdown" mapstructure:"send_markdown"`
	AllowFrom          models.Allow `json:"allow_from" mapstructure:"allow_from"`
}

// ParseConfig 解析渠道配置
func ParseConfig(config map[string]any) (Config, error) {
	cfg := Config{}
	data, err := json.Marshal(config)
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return cfg, err
	}

	if v, ok := config["allow_from"]; ok {
		if arr, ok := v.([]any); ok {
			for _, item := range arr {
				if s, ok := item.(string); ok {
					cfg.AllowFrom = append(cfg.AllowFrom, s)
				}
			}
		} else if s, ok := v.(string); ok && s != "" {
			for _, part := range strings.Split(s, ",") {
				part = strings.TrimSpace(part)
				if part != "" {
					cfg.AllowFrom = append(cfg.AllowFrom, part)
				}
			}
		}
	}

	if cfg.MaxMessageLength == 0 {
		cfg.MaxMessageLength = 2000
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.AppID == "" || c.AppSecret == "" {
		return fmt.Errorf("qq app_id and app_secret are required")
	}
	return nil
}
