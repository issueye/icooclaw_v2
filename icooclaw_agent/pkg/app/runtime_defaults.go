package app

import "icooclaw/pkg/script"

func scriptDefaultTimeoutSeconds() int {
	cfg := script.DefaultConfig()
	return cfg.ExecTimeout
}
