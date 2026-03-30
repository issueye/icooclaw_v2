package storage

import (
	"strings"

	"icooclaw/pkg/consts"
)

// ResolveDefaultAgent returns the configured default agent when available,
// otherwise falls back to the built-in "default" agent record.
func (s *Storage) ResolveDefaultAgent() (*Agent, error) {
	if s == nil || s.agent == nil {
		return nil, nil
	}

	if s.param != nil {
		cfg, err := s.param.Get(consts.DEFAULT_AGENT_ID_KEY)
		if err != nil {
			return nil, err
		}
		if cfg != nil {
			agentID := strings.TrimSpace(cfg.Value)
			if agentID != "" {
				agentInfo, err := s.agent.GetByID(agentID)
				if err == nil && agentInfo != nil && agentInfo.Type == AgentTypeMaster {
					return agentInfo, nil
				}
			}
		}
	}

	return s.agent.GetDefault()
}
