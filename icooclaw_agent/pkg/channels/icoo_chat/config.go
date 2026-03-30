package icoo_chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"icooclaw/pkg/channels/models"
)

type Config struct {
	Enabled   bool         `json:"enabled" mapstructure:"enabled"`
	Endpoint  string       `json:"endpoint" mapstructure:"endpoint"`
	AppID     string       `json:"app_id" mapstructure:"app_id"`
	AppSecret string       `json:"app_secret" mapstructure:"app_secret"`
	AllowFrom models.Allow `json:"allow_from" mapstructure:"allow_from"`
}

func ParseConfig(config map[string]any) (Config, error) {
	cfg := Config{}
	data, err := json.Marshal(config)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	if v, ok := config["allow_from"]; ok {
		if arr, ok := v.([]any); ok {
			for _, item := range arr {
				if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
					cfg.AllowFrom = append(cfg.AllowFrom, strings.TrimSpace(s))
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

	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Endpoint) == "" {
		return fmt.Errorf("icoo_chat endpoint is required")
	}
	if strings.TrimSpace(c.AppID) == "" || strings.TrimSpace(c.AppSecret) == "" {
		return fmt.Errorf("icoo_chat app_id and app_secret are required")
	}
	return nil
}
