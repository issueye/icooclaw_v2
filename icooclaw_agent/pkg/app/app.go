package app

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"icooclaw/pkg/agent"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels"
	"icooclaw/pkg/config"
	"icooclaw/pkg/mcp"
	"icooclaw/pkg/memory"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/scheduler"
	"icooclaw/pkg/skill"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
	"log/slog"

	"github.com/go-chi/chi/v5"
)

type App struct {
	Ctx             context.Context
	Cancel          context.CancelFunc
	Logger          *slog.Logger
	LogCloser       io.Closer
	Cfg             *config.Config
	Storage         *storage.Storage
	MessageBus      *bus.MessageBus
	ProviderManager *providers.Manager
	DefaultProvider providers.Provider
	ToolRegistry    *tools.Registry
	MCPManager      *mcp.Manager
	MemoryLoader    memory.Loader
	SkillLoader     skill.Loader
	AgentManager    *agent.AgentManager
	ChannelManager  *channels.Manager
	Gw              gatewayServer
	Scheduler       *scheduler.Scheduler
}

type gatewayServer interface {
	Start() error
	Shutdown(ctx context.Context) error
	Router() chi.Router
}

func NewApp() *App {
	return &App{}
}

func (a *App) Init(path string) error {
	a.Ctx, a.Cancel = context.WithCancel(context.Background())

	if err := a.InitConfig(path); err != nil {
		return err
	}
	a.Logger = a.InitLog()
	a.InitStorage()
	if err := a.InitRuntime(); err != nil {
		return err
	}
	a.InitGateway()
	return nil
}

// RunGateway 运行网关服务
func (a *App) RunGateway() {
	go func() {
		err := a.ChannelManager.Start(a.Ctx)
		if err != nil {
			a.Logger.Error("渠道管理器错误", "error", err)
		}
	}()

	go func() {
		err := a.AgentManager.Start(a.Ctx)
		if err != nil {
			a.Logger.Error("智能体管理器启动失败", "error", err)
		}
	}()

	a.Scheduler.Start()

	err := a.Gw.Start()
	if err != nil && err != http.ErrServerClosed {
		a.Logger.Error("网关服务错误", "error", err)
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		a.Logger.Info("正在关闭网关服务...")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		err := a.ChannelManager.Stop(shutdownCtx)
		if err != nil {
			a.Logger.Error("关闭渠道管理器失败", "error", err)
		}

		err = a.Gw.Shutdown(shutdownCtx)
		if err != nil {
			a.Logger.Error("网关服务关闭失败", "error", err)
		}

		if a.MCPManager != nil {
			if err := a.MCPManager.Close(); err != nil {
				a.Logger.Error("关闭MCP管理器失败", "error", err)
			}
		}

		if a.LogCloser != nil {
			if err := a.LogCloser.Close(); err != nil {
				a.Logger.Error("关闭日志输出失败", "error", err)
			}
		}

		a.Cancel()
	}()
}
