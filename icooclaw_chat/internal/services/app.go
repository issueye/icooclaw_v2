package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	app := &App{}
	return app
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	configService := GetConfigService()
	configService.Init(ctx)
	GetAgentProcessManager().Init(ctx)

	clawCfg := configService.GetClawConnectionConfig()
	if clawCfg.APIBase != "" {
		GetAPIProxy().SetTargetBase(clawCfg.APIBase)
	}
}

func (a *App) Greet(name string) string {
	return fmt.Sprintf("你好 %s，现在是展示时间！", name)
}

func (a *App) MinimizeWindow() {
	runtime.WindowMinimise(a.ctx)
}

func (a *App) CloseWindow() {
	GetAgentProcessManager().Shutdown()
	runtime.Quit(a.ctx)
}

func (a *App) Shutdown(ctx context.Context) {
	GetAgentProcessManager().Shutdown()
}

func (a *App) GetClawConnectionConfig() map[string]string {
	cfg := GetConfigService().GetClawConnectionConfig()
	agentCfg := GetConfigService().GetAgentProcessConfig()
	return map[string]string{
		"apiBase":   cfg.APIBase,
		"wsHost":    cfg.WSHost,
		"wsPort":    cfg.WSPort,
		"wsPath":    cfg.WSPath,
		"userId":    cfg.UserID,
		"agentPath": agentCfg.BinaryPath,
	}
}

func (a *App) SetClawConnectionConfig(apiBase, wsHost, wsPort, wsPath, userId, agentPath string) error {
	cfg := ClawConnectionConfig{
		APIBase: apiBase,
		WSHost:  wsHost,
		WSPort:  wsPort,
		WSPath:  wsPath,
		UserID:  userId,
	}
	GetAPIProxy().SetTargetBase(apiBase)
	if err := GetConfigService().SetClawConnectionConfig(cfg); err != nil {
		return err
	}
	return GetConfigService().SetAgentProcessConfig(AgentProcessConfig{
		BinaryPath: strings.TrimSpace(agentPath),
	})
}

func (a *App) GetAgentProcessStatus() AgentProcessStatus {
	return GetAgentProcessManager().Status()
}

func (a *App) WakeAgent() (AgentProcessStatus, error) {
	return GetAgentProcessManager().Wake()
}

func (a *App) StopAgent() (AgentProcessStatus, error) {
	return GetAgentProcessManager().Stop()
}

func (a *App) RestartAgent() (AgentProcessStatus, error) {
	return GetAgentProcessManager().Restart()
}
