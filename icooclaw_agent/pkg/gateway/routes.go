package gateway

import (
	"log/slog"
	"net/http"

	"icooclaw/pkg/agent"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/gateway/handlers"
	"icooclaw/pkg/mcp"
	"icooclaw/pkg/scheduler"
	"icooclaw/pkg/storage"

	"github.com/go-chi/chi/v5"
)

type standardResourceRoutes interface {
	Page(http.ResponseWriter, *http.Request)
	Create(http.ResponseWriter, *http.Request)
	Update(http.ResponseWriter, *http.Request)
	Delete(http.ResponseWriter, *http.Request)
	GetByID(http.ResponseWriter, *http.Request)
	GetAll(http.ResponseWriter, *http.Request)
	GetEnabled(http.ResponseWriter, *http.Request)
}

// Handlers 封装所有处理器
type Handlers struct {
	Schedule  *scheduler.Scheduler
	Common    *handlers.CommonHandler
	Session   *handlers.SessionHandler
	Message   *handlers.MessageHandler
	MCP       *handlers.MCPHandler
	Memory    *handlers.MemoryHandler
	Task      *handlers.TaskHandler
	Provider  *handlers.ProviderHandler
	Agent     *handlers.AgentHandler
	Skill     *handlers.SkillHandler
	Channel   *handlers.ChannelHandler
	Param     *handlers.ParamHandler
	Tool      *handlers.ToolHandler
	Workspace *handlers.WorkspaceHandler
}

// NewHandlers 创建所有处理器
func NewHandlers(
	logger *slog.Logger,
	storage *storage.Storage,
	schedule *scheduler.Scheduler,
	agentManager *agent.AgentManager,
	bus *bus.MessageBus,
	mcpManager *mcp.Manager,
) *Handlers {

	return &Handlers{
		Schedule:  schedule,
		Common:    handlers.NewCommonHandler(logger),
		Session:   handlers.NewSessionHandler(logger, storage),
		Message:   handlers.NewMessageHandler(logger, storage),
		MCP:       handlers.NewMCPHandler(logger, storage, mcpManager),
		Memory:    handlers.NewMemoryHandler(logger, storage),
		Task:      handlers.NewTaskHandler(logger, storage, schedule),
		Provider:  handlers.NewProviderHandler(logger, storage),
		Agent:     handlers.NewAgentHandler(logger, storage),
		Skill:     handlers.NewSkillHandler(logger, storage),
		Channel:   handlers.NewChannelHandler(logger, storage),
		Param:     handlers.NewParamHandler(logger, storage),
		Tool:      handlers.NewToolHandler(logger, storage),
		Workspace: handlers.NewWorkspaceHandler(logger, storage, agentManager),
	}
}

// RegisterRoutes 注册所有 CRUD 路由
func RegisterRoutes(r chi.Router, h *Handlers) {
	// 健康检查
	r.Get("/api/v1/health", h.Common.HealthCheck)

	// Session 路由
	r.Route("/api/v1/sessions", func(r chi.Router) {
		r.Post("/page", h.Session.Page)     // 分页查询
		r.Post("/save", h.Session.Save)     // 保存
		r.Post("/create", h.Session.Create) // 创建新会话
		r.Post("/delete", h.Session.Delete) // 删除
		r.Post("/get", h.Session.GetByID)   // 获取单个
	})

	// Message 路由
	r.Route("/api/v1/messages", func(r chi.Router) {
		r.Post("/page", h.Message.Page)
		r.Post("/create", h.Message.Create)
		r.Post("/update", h.Message.Update)
		r.Post("/delete", h.Message.Delete)
		r.Post("/get", h.Message.GetByID)
		r.Post("/by-session", h.Message.GetBySessionID)
	})

	// MCP 路由
	r.Route("/api/v1/mcp", func(r chi.Router) {
		r.Post("/page", h.MCP.Page)
		r.Post("/create", h.MCP.Create)
		r.Post("/update", h.MCP.Update)
		r.Post("/delete", h.MCP.Delete)
		r.Post("/get", h.MCP.GetByID)
		r.Post("/runtime/connect", h.MCP.Connect)
		r.Get("/runtime/all", h.MCP.GetRuntimeAll)
		r.Get("/all", h.MCP.GetAll)
	})

	// Memory 路由
	r.Route("/api/v1/memories", func(r chi.Router) {
		r.Post("/page", h.Memory.Page)
		r.Post("/create", h.Memory.Create)
		r.Post("/update", h.Memory.Update)
		r.Post("/delete", h.Memory.Delete)
		r.Post("/get", h.Memory.GetByID)
		r.Post("/search", h.Memory.Search)
	})

	// Task 路由
	r.Route("/api/v1/tasks", func(r chi.Router) {
		r.Post("/page", h.Task.Page)
		r.Post("/create", h.Task.Create)
		r.Post("/update", h.Task.Update)
		r.Post("/delete", h.Task.Delete)
		r.Post("/get", h.Task.GetByID)
		r.Post("/toggle", h.Task.ToggleEnabled)
		r.Post("/execute", h.Task.Execute)
		r.Get("/all", h.Task.GetAll)
		r.Get("/enabled", h.Task.GetEnabled)
	})

	registerStandardResourceRoutes(r, "/api/v1/providers", h.Provider)
	registerStandardResourceRoutes(r, "/api/v1/agents", h.Agent)

	// Skill 路由
	r.Route("/api/v1/skills", func(r chi.Router) {
		r.Post("/page", h.Skill.Page)
		r.Post("/create", h.Skill.Create)
		r.Post("/update", h.Skill.Update)
		r.Post("/delete", h.Skill.Delete)
		r.Post("/get", h.Skill.GetByID)
		r.Post("/get-by-name", h.Skill.GetByName)
		r.Post("/upsert", h.Skill.Upsert)
		r.Post("/install", h.Skill.Install)
		r.Post("/import", h.Skill.Import)
		r.Get("/export", h.Skill.Export)
		r.Get("/all", h.Skill.GetAll)
		r.Get("/enabled", h.Skill.GetEnabled)
	})

	registerStandardResourceRoutes(r, "/api/v1/channels", h.Channel)

	// 参数配置路由
	r.Route("/api/v1/params", func(r chi.Router) {
		r.Post("/page", h.Param.Page)           // 分页查询
		r.Post("/create", h.Param.Create)       // 创建
		r.Post("/update", h.Param.Update)       // 更新
		r.Post("/delete", h.Param.Delete)       // 删除
		r.Post("/get", h.Param.GetByID)         // 通过 ID 获取
		r.Post("/get-by-key", h.Param.GetByKey) // 通过键获取
		r.Get("/all", h.Param.GetAll)           // 获取所有
		r.Post("/by-group", h.Param.GetByGroup) // 按分组获取

		// 便捷接口
		r.Post("/default-model/set", h.Param.SetDefaultModel) // 设置默认模型
		r.Get("/default-model/get", h.Param.GetDefaultModel)  // 获取默认模型
		r.Post("/default-agent/set", h.Param.SetDefaultAgent) // 设置默认智能体
		r.Get("/default-agent/get", h.Param.GetDefaultAgent)  // 获取默认智能体
		r.Post("/exec-env/set", h.Param.SetExecEnv)           // 设置命令执行环境变量
		r.Get("/exec-env/get", h.Param.GetExecEnv)            // 获取命令执行环境变量
	})

	registerStandardResourceRoutes(r, "/api/v1/tools", h.Tool)

	r.Route("/api/v1/workspace", func(r chi.Router) {
		r.Post("/prompt/get", h.Workspace.GetPrompt)
		r.Post("/prompt/save", h.Workspace.SavePrompt)
		r.Post("/prompt/generate", h.Workspace.GeneratePrompt)
	})
}

func registerStandardResourceRoutes(r chi.Router, pattern string, handler standardResourceRoutes) {
	r.Route(pattern, func(r chi.Router) {
		r.Post("/page", handler.Page)
		r.Post("/create", handler.Create)
		r.Post("/update", handler.Update)
		r.Post("/delete", handler.Delete)
		r.Post("/get", handler.GetByID)
		r.Get("/all", handler.GetAll)
		r.Get("/enabled", handler.GetEnabled)
	})
}
