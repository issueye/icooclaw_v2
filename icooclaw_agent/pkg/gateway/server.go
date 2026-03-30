package gateway

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"icooclaw/pkg/agent"
	"icooclaw/pkg/bus"
	authmw "icooclaw/pkg/gateway/middleware"
	"icooclaw/pkg/mcp"
	"icooclaw/pkg/scheduler"
	"icooclaw/pkg/storage"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// Server 表示网关 HTTP 服务。
type Server struct {
	router       chi.Router
	server       *http.Server
	cfg          *ServerConfig
	storage      *storage.Storage
	logger       *slog.Logger
	handlers     *Handlers
	schedule     *scheduler.Scheduler
	bus          *bus.MessageBus
	agentManager *agent.AgentManager
}

type Dependencies struct {
	Config       *ServerConfig
	Logger       *slog.Logger
	Storage      *storage.Storage
	Scheduler    *scheduler.Scheduler
	Bus          *bus.MessageBus
	AgentManager *agent.AgentManager
	MCPManager   *mcp.Manager
}

// ServerConfig 保存网关服务配置。
type ServerConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	MaxConcurrentWS int
	Security        SecurityConfig
}

// DefaultServerConfig 返回默认网关服务配置。
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Addr:            ":16789",
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     60 * time.Second,
		MaxConcurrentWS: 100,
		Security:        DefaultSecurityConfig(),
	}
}

// NewServer 创建一个新的网关服务。
func NewServer(deps Dependencies) *Server {
	if deps.Config == nil {
		deps.Config = DefaultServerConfig()
	}
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}

	r := chi.NewRouter()
	s := &Server{
		router:       r,
		cfg:          deps.Config,
		storage:      deps.Storage,
		logger:       deps.Logger,
		schedule:     deps.Scheduler,
		bus:          deps.Bus,
		agentManager: deps.AgentManager,
	}

	// 创建处理器
	s.handlers = NewHandlers(
		deps.Logger,
		deps.Storage,
		deps.Scheduler,
		s.agentManager,
		s.bus,
		deps.MCPManager,
	)

	// 初始化中间件
	s.setupMiddleware(deps.Config)

	// 注册路由
	RegisterRoutes(r, s.handlers)

	// 创建 HTTP 服务
	s.server = &http.Server{
		Addr:         deps.Config.Addr,
		Handler:      r,
		ReadTimeout:  deps.Config.ReadTimeout,
		WriteTimeout: deps.Config.WriteTimeout,
		IdleTimeout:  deps.Config.IdleTimeout,
	}

	return s
}

// setupMiddleware 初始化中间件链。
func (s *Server) setupMiddleware(cfg *ServerConfig) {
	// 请求 ID
	s.router.Use(chimiddleware.RequestID)

	// 真实 IP
	s.router.Use(chimiddleware.RealIP)

	// 结构化请求日志
	s.router.Use(authmw.RequestLogger(s.logger))

	// 异常恢复
	s.router.Use(chimiddleware.Recoverer)

	// 跨域配置
	s.router.Use(corsMiddleware(cfg.Security))

	// 安全校验
	s.router.Use(s.securityMiddleware(cfg.Security))
}

// corsMiddleware 处理跨域响应头。
func corsMiddleware(security SecurityConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if security.AllowsRequest(r) {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Add("Vary", "Origin")
				}
			} else {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

			if r.Method == http.MethodOptions {
				if origin != "" && !security.AllowsRequest(r) {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) securityMiddleware(security SecurityConfig) func(http.Handler) http.Handler {
	if strings.EqualFold(strings.TrimSpace(security.Mode), SecurityModeOpen) {
		return func(next http.Handler) http.Handler { return next }
	}

	authenticator := security.Authenticator()
	required := authmw.RequireAuth(authenticator, s.logger)

	return func(next http.Handler) http.Handler {
		protected := required(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions || r.URL.Path == "/api/v1/health" {
				next.ServeHTTP(w, r)
				return
			}
			protected.ServeHTTP(w, r)
		})
	}
}

// Start 启动 HTTP 服务。
func (s *Server) Start() error {
	s.logger.Info("已启动", "addr", s.server.Addr)

	err := s.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		s.logger.Error("【网关服务】已启动失败", "error", err)
	}
	return nil
}

// Shutdown 优雅关闭服务。
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("正在关闭")

	return s.server.Shutdown(ctx)
}

// Router 返回 chi 路由器。
func (s *Server) Router() chi.Router {
	return s.router
}

// Bus 返回消息总线。
func (s *Server) Bus() *bus.MessageBus {
	return s.bus
}
