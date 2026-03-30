package app

import (
	"fmt"

	"icooclaw/pkg/channels"
	"icooclaw/pkg/gateway"
	"icooclaw/pkg/logging"
)

// InitGateway 初始化网关服务器
func (a *App) InitGateway() {
	serverCfg := gateway.DefaultServerConfig()
	if a.Cfg.Gateway.Port > 0 {
		serverCfg.Addr = fmt.Sprintf(":%d", a.Cfg.Gateway.Port)
	}
	serverCfg.Security = gateway.SecurityConfig{
		Mode:   a.Cfg.Gateway.Security.Mode,
		APIKey: a.Cfg.Gateway.Security.APIKey,
	}

	a.Gw = gateway.NewServer(gateway.Dependencies{
		Config:       serverCfg,
		Logger:       logging.Component(a.Logger, "gateway"),
		Storage:      a.Storage,
		Scheduler:    a.Scheduler,
		Bus:          a.MessageBus,
		AgentManager: a.AgentManager,
		MCPManager:   a.MCPManager,
	})

	// 初始化渠道管理器
	a.ChannelManager = channels.NewManager(a.Logger, a.Gw.Router(), a.MessageBus, a.Storage.Channel())
}
