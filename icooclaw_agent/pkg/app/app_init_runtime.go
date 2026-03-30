package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"icooclaw/pkg/agent"
	"icooclaw/pkg/agent/react"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/logging"
	icmcp "icooclaw/pkg/mcp"
	"icooclaw/pkg/memory"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/scheduler"
	schedulerTool "icooclaw/pkg/scheduler/tool"
	"icooclaw/pkg/skill"
	skillTool "icooclaw/pkg/skill/tool"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
	"icooclaw/pkg/tools/builtin"
)

func (a *App) InitRuntime() error {
	a.initMessageBus()
	err := a.initScheduler()
	if err != nil {
		return err
	}

	a.InitTool()
	a.InitMemory()
	a.InitSkill()
	err = skill.SyncWorkspaceSkills(a.Cfg.Workspace, a.Storage.Skill())
	if err != nil {
		return fmt.Errorf("sync workspace skills failed: %w", err)
	}

	a.InitProvider()
	a.InitAgentTools()
	a.InitSkillTools()
	a.InitMCP()

	summaryAgent, err := a.buildSummaryAgent()
	if err != nil {
		return fmt.Errorf("create summary agent failed: %w", err)
	}

	hooks := a.buildHooks(summaryAgent)
	a.Scheduler.RegisterExecutor(scheduler.TaskExecutorSummary, hooks.ExecuteSummaryTask)
	a.AgentManager, err = a.buildAgentManager(hooks)
	if err != nil {
		return fmt.Errorf("create agent manager failed: %w", err)
	}

	return nil
}

// InitMCP 初始化 MCP 管理器并加载启用配置。
func (a *App) InitMCP() {
	a.MCPManager = icmcp.NewManager(
		a.ToolRegistry,
		icmcp.WithManagerLogger(logging.Component(a.Logger, "mcp")),
	)

	configs, err := a.Storage.MCP().ListEnabled()
	if err != nil {
		a.Logger.Error("加载MCP配置失败", "error", err)
		return
	}

	for _, cfg := range configs {
		if err := a.connectMCPConfig(cfg); err != nil {
			a.Logger.Warn("连接MCP失败", "name", cfg.Name, "error", err)
		}
	}
}

func (a *App) connectMCPConfig(cfg *storage.MCPConfig) error {
	if cfg == nil {
		return fmt.Errorf("mcp config is nil")
	}

	client := a.MCPManager.CreateAndAddClient(
		cfg.Name,
		icmcp.WithLogger(logging.Component(a.Logger, "mcp")),
		icmcp.WithRetryConfig(normalizeMCPRetryCount(cfg.RetryCount), 2*time.Second),
	)

	timeout := time.Duration(normalizeMCPTimeoutSeconds(cfg.TimeoutSeconds)) * time.Second
	ctx, cancel := context.WithTimeout(a.Ctx, timeout)
	defer cancel()

	switch {
	case cfg.IsStdio():
		if strings.TrimSpace(cfg.Command) == "" {
			_ = a.MCPManager.RemoveClient(cfg.Name)
			return fmt.Errorf("stdio MCP 缺少 command")
		}
		if err := client.ConnectStdio(ctx, cfg.Command, cfg.Args, cfg.Env); err != nil {
			_ = a.MCPManager.RemoveClient(cfg.Name)
			return err
		}
	case cfg.IsSSE():
		if strings.TrimSpace(cfg.URL) == "" {
			_ = a.MCPManager.RemoveClient(cfg.Name)
			return fmt.Errorf("sse MCP 缺少 url")
		}
		if err := client.ConnectSSE(ctx, cfg.URL, cfg.Headers); err != nil {
			_ = a.MCPManager.RemoveClient(cfg.Name)
			return err
		}
	default:
		_ = a.MCPManager.RemoveClient(cfg.Name)
		return fmt.Errorf("不支持的 MCP 类型: %s", cfg.Type)
	}

	// 连接成功后重新注册工具，确保远端工具列表已经被发现。
	a.MCPManager.AddClient(cfg.Name, client)
	a.Logger.Info("MCP连接成功", "name", cfg.Name, "type", cfg.Type, "tools", client.GetToolNames())
	return nil
}

func normalizeMCPRetryCount(retryCount int) int {
	if retryCount <= 0 {
		return 3
	}
	return retryCount
}

func normalizeMCPTimeoutSeconds(timeoutSeconds int) int {
	if timeoutSeconds <= 0 {
		return scriptDefaultTimeoutSeconds()
	}
	return timeoutSeconds
}

func (a *App) initMessageBus() {
	a.MessageBus = bus.NewMessageBus(bus.DefaultConfig())
	a.MessageBus.SetLogger(logging.Component(a.Logger, "bus"))
	a.MessageBus.SetAlertHandler(func(alert bus.Alert) {
		logging.Component(a.Logger, "bus").Warn(
			"bus alert",
			"type", alert.Type,
			"message", alert.Message,
			"inbound_utilization", alert.Metrics.InboundUtilization,
			"outbound_utilization", alert.Metrics.OutboundUtilization,
			"drop_count", alert.Metrics.DropCount,
		)
	})
	a.MessageBus.StartMetricsReporter(a.Ctx)
}

func (a *App) initScheduler() error {
	a.Scheduler = scheduler.NewScheduler(
		a.Storage.Task(),
		a.MessageBus,
		logging.Component(a.Logger, "scheduler"),
	)

	if err := a.Scheduler.LoadTasks(); err != nil {
		return fmt.Errorf("load scheduler tasks failed: %w", err)
	}

	return nil
}

func (a *App) buildSummaryAgent() (*react.ReActAgent, error) {
	return react.NewReActAgentNoHooks(a.Ctx, react.Dependencies{
		ProviderManager: a.ProviderManager,
		Storage:         a.Storage,
		Logger:          a.Logger,
	})
}

func (a *App) buildHooks(summaryAgent *react.ReActAgent) *AppAgentHooks {
	return NewAppAgentHooks(HooksDependencies{
		Logger:       a.Logger,
		Workspace:    a.Cfg.Workspace,
		Storage:      a.Storage,
		Scheduler:    a.Scheduler,
		SummaryAgent: summaryAgent,
		RecentCount:  a.Cfg.Agent.RecentCount,
	})
}

func (a *App) buildAgentManager(hooks react.ReactHooks) (*agent.AgentManager, error) {
	return agent.NewAgentManager(agent.Dependencies{
		Logger:            a.Logger,
		Bus:               a.MessageBus,
		Memory:            a.MemoryLoader,
		Skills:            a.SkillLoader,
		Tools:             a.ToolRegistry,
		Hooks:             hooks,
		ProviderManager:   a.ProviderManager,
		Storage:           a.Storage,
		MaxToolIterations: consts.DEFAULT_TOOL_ITERATIONS,
	})
}

// InitAgentTools 初始化 agent 自身使用的工具
func (a *App) InitAgentTools() {
	subAgentTool := builtin.NewSubAgentTool(
		a.Storage,
		a.Scheduler,
		a.ProviderManager,
		a.SkillLoader,
		a.ToolRegistry,
		a.Logger,
	)
	a.Scheduler.RegisterExecutor(scheduler.TaskExecutorSubAgent, builtin.NewSubAgentTaskExecutor(
		a.Storage,
		a.ProviderManager,
		a.SkillLoader,
		a.ToolRegistry,
		a.MessageBus,
		a.Logger,
	))
	a.ToolRegistry.Register(subAgentTool)
}

// InitTool 初始化工具，包括内置工具
func (a *App) InitTool() {
	a.ToolRegistry = tools.NewRegistry()

	builtin.RegisterBuiltinTools(a.ToolRegistry, a.Storage, a.Cfg)

	schedulers := schedulerTool.RegisterTaskTools(a.Storage.Task(), a.Scheduler, a.MessageBus, a.Logger)
	for _, t := range schedulers {
		a.ToolRegistry.Register(t)
	}

	skillInstallTool := skillTool.NewInstallSkillTool(
		a.Cfg.Workspace,
		a.Cfg.ClawHubAddr,
		a.Logger,
		a.Storage.Skill(),
		time.Minute,
	)
	a.ToolRegistry.Register(skillInstallTool)

	skillSearchTool := skillTool.NewSearchSkillTool(
		a.Cfg.Workspace,
		a.Cfg.ClawHubAddr,
		a.Logger,
		a.Storage.Skill(),
		time.Minute,
	)
	a.ToolRegistry.Register(skillSearchTool)
}

// InitProvider 初始化提供商工厂
func (a *App) InitProvider() {
	manager := providers.NewManager(a.Storage, a.Logger)

	var defaultProvider providers.Provider
	defaultModel, err := a.Storage.Param().Get(consts.DEFAULT_MODEL_KEY)
	if err != nil || defaultModel == nil || defaultModel.Value == "" {
		a.Logger.Warn("未找到默认模型，需要配置", "key", consts.DEFAULT_MODEL_KEY)
	} else {
		arrs := strings.Split(defaultModel.Value, "/")
		if len(arrs) != 2 {
			a.Logger.Warn("默认模型格式错误，需要配置", "model", defaultModel.Value)
			return
		}

		a.Logger.Info("默认模型", "model", defaultModel.Value)
		defaultProvider, err = manager.Get(arrs[0])
		if err != nil {
			a.Logger.Warn("未找到默认提供商，需要配置", "provider", arrs[0])
		}
	}

	a.DefaultProvider = defaultProvider
	a.ProviderManager = manager
}

// InitMemory 初始化记忆加载器
func (a *App) InitMemory() {
	recentCount := a.Cfg.Agent.RecentCount
	if recentCount <= 0 {
		recentCount = consts.DefaultRecentCount
	}
	a.MemoryLoader = memory.NewLoaderWithRecentCount(a.Storage, 100, recentCount, logging.Component(a.Logger, "memory"))
}

// InitSkill 初始化 skill 加载器
func (a *App) InitSkill() {
	a.SkillLoader = skill.NewLoader(a.Cfg.Workspace, a.Storage, logging.Component(a.Logger, "skill"))
}

// InitSkillTools 初始化技能执行工具
func (a *App) InitSkillTools() {
	skillListTool := skillTool.NewListSkillsTool(a.Storage, a.Logger)
	a.ToolRegistry.Register(skillListTool)

	skillInfoTool := skillTool.NewGetSkillInfoTool(a.Storage, a.Logger)
	a.ToolRegistry.Register(skillInfoTool)

	if a.ProviderManager == nil {
		return
	}

	skillExecTool := skillTool.NewExecuteSkillTool(
		a.Cfg.Workspace,
		a.Storage,
		a.ProviderManager,
		a.ToolRegistry,
		a.Logger,
	)
	a.ToolRegistry.Register(skillExecTool)
}
