package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"icooclaw/pkg/gateway/models"
	icmcp "icooclaw/pkg/mcp"
	"icooclaw/pkg/storage"
)

type MCPHandler struct {
	*standardResourceHandler[*storage.MCPConfig, *storage.QueryMCP, *storage.ResQueryMCP]
	logger  *slog.Logger
	storage *storage.Storage
	manager *icmcp.Manager
}

type MCPRuntimeInfo struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	State         string   `json:"state"`
	Tools         []string `json:"tools"`
	ToolCount     int      `json:"tool_count"`
	LastError     string   `json:"last_error,omitempty"`
	LastErrorAt   string   `json:"last_error_at,omitempty"`
	Managed       bool     `json:"managed"`
	Enabled       bool     `json:"enabled"`
	Connected     bool     `json:"connected"`
	RuntimeLoaded bool     `json:"runtime_loaded"`
}

func NewMCPHandler(logger *slog.Logger, st *storage.Storage, manager *icmcp.Manager) *MCPHandler {
	if st == nil {
		return &MCPHandler{logger: logger, storage: st, manager: manager}
	}

	validate := func(cfg *storage.MCPConfig) error {
		return normalizeMCPConfig(cfg)
	}

	return &MCPHandler{
		logger:  logger,
		storage: st,
		manager: manager,
		standardResourceHandler: &standardResourceHandler[*storage.MCPConfig, *storage.QueryMCP, *storage.ResQueryMCP]{
			logger: logger,
			messages: resourceMessages{
				BindPage:      "绑定分页请求失败",
				PageFailed:    "获取MCP配置列表失败",
				PageSuccess:   "MCP配置列表获取成功",
				BindCreate:    "绑定创建MCP配置请求失败",
				CreateFailed:  "创建MCP配置失败",
				CreateSuccess: "MCP配置创建成功",
				BindUpdate:    "绑定更新MCP配置请求失败",
				UpdateFailed:  "更新MCP配置失败",
				UpdateSuccess: "MCP配置更新成功",
				BindDelete:    "绑定删除MCP配置请求失败",
				DeleteFailed:  "删除MCP配置失败",
				DeleteSuccess: "MCP配置删除成功",
				BindGet:       "绑定获取MCP配置请求失败",
				GetFailed:     "获取MCP配置失败",
				GetSuccess:    "MCP配置获取成功",
				GetAllFailed:  "获取所有MCP配置失败",
				GetAllSuccess: "MCP配置列表获取成功",
			},
			page:         st.MCP().Page,
			create:       st.MCP().Create,
			update:       st.MCP().Update,
			delete:       st.MCP().DeleteByID,
			get:          st.MCP().GetByID,
			list:         st.MCP().List,
			beforeCreate: validate,
			beforeUpdate: validate,
		},
	}
}

func normalizeMCPType(value storage.MCPType) storage.MCPType {
	normalized := strings.TrimSpace(strings.ToLower(value.String()))
	switch normalized {
	case "", storage.MCPTypeStdio.String():
		return storage.MCPTypeStdio
	case storage.MCPTypeSSE.String(), "streamable http":
		return storage.MCPTypeSSE
	default:
		return storage.MCPType(normalized)
	}
}

func normalizeMCPConfig(cfg *storage.MCPConfig) error {
	if cfg == nil {
		return fmt.Errorf("mcp config is nil")
	}

	cfg.Name = strings.TrimSpace(cfg.Name)
	cfg.Description = strings.TrimSpace(cfg.Description)
	cfg.Command = strings.TrimSpace(cfg.Command)
	cfg.URL = strings.TrimSpace(cfg.URL)
	cfg.Type = normalizeMCPType(cfg.Type)

	if cfg.RetryCount <= 0 {
		cfg.RetryCount = 3
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 30
	}
	if cfg.Args == nil {
		cfg.Args = storage.StringArray{}
	}
	if cfg.Env == nil {
		cfg.Env = map[string]string{}
	}
	if cfg.Headers == nil {
		cfg.Headers = map[string]string{}
	}

	switch cfg.Type {
	case storage.MCPTypeStdio:
		cfg.URL = ""
		cfg.Headers = map[string]string{}
		if cfg.Command == "" {
			return fmt.Errorf("stdio MCP 缺少 command")
		}
	case storage.MCPTypeSSE:
		cfg.Command = ""
		cfg.Args = storage.StringArray{}
		cfg.Env = map[string]string{}
		if cfg.URL == "" {
			return fmt.Errorf("sse MCP 缺少 url")
		}
	default:
		return fmt.Errorf("不支持的 MCP 类型: %s", cfg.Type)
	}

	if cfg.Name == "" {
		return fmt.Errorf("mcp 名称不能为空")
	}

	return nil
}

func managedMCPTimeout(cfg *storage.MCPConfig) time.Duration {
	timeoutSeconds := cfg.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}
	return time.Duration(timeoutSeconds) * time.Second
}

func managedMCPRetryCount(cfg *storage.MCPConfig) int {
	if cfg.RetryCount <= 0 {
		return 3
	}
	return cfg.RetryCount
}

func (h *MCPHandler) connectWithManager(ctx context.Context, cfg *storage.MCPConfig) (*icmcp.Client, error) {
	if h.manager == nil {
		return nil, fmt.Errorf("mcp manager is not initialized")
	}

	_ = h.manager.RemoveClient(cfg.Name)
	client := h.manager.CreateAndAddClient(
		cfg.Name,
		icmcp.WithLogger(h.logger.With("mcp_name", cfg.Name)),
		icmcp.WithRetryConfig(managedMCPRetryCount(cfg), 2*time.Second),
	)

	switch cfg.Type {
	case storage.MCPTypeStdio:
		if err := client.ConnectStdio(ctx, cfg.Command, cfg.Args, cfg.Env); err != nil {
			_ = h.manager.RemoveClient(cfg.Name)
			return nil, err
		}
	case storage.MCPTypeSSE:
		if err := client.ConnectSSE(ctx, cfg.URL, cfg.Headers); err != nil {
			_ = h.manager.RemoveClient(cfg.Name)
			return nil, err
		}
	default:
		_ = h.manager.RemoveClient(cfg.Name)
		return nil, fmt.Errorf("不支持的 MCP 类型: %s", cfg.Type)
	}

	h.manager.AddClient(cfg.Name, client)
	return client, nil
}

func (h *MCPHandler) runtimeInfoFor(cfg *storage.MCPConfig) MCPRuntimeInfo {
	info := MCPRuntimeInfo{
		ID:      cfg.ID,
		Name:    cfg.Name,
		State:   icmcp.ConnectionStateDisconnected.String(),
		Tools:   []string{},
		Managed: false,
		Enabled: cfg.Enabled,
	}

	if h.manager == nil {
		return info
	}

	client := h.manager.GetClient(cfg.Name)
	if client == nil {
		return info
	}

	lastErr, lastErrAt := client.GetLastError()
	info.Managed = true
	info.RuntimeLoaded = true
	info.State = client.GetState().String()
	info.Connected = client.IsConnected()
	info.Tools = client.GetToolNames()
	info.ToolCount = len(info.Tools)
	if lastErr != nil {
		info.LastError = lastErr.Error()
	}
	if !lastErrAt.IsZero() {
		info.LastErrorAt = lastErrAt.Format(time.RFC3339)
	}
	return info
}

// Save 保存MCP配置
func (h *MCPHandler) Save(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.MCPConfig](r)
	if err != nil {
		h.logger.Error("绑定保存MCP配置请求失败", "error", err)
		http.Error(w, "绑定保存MCP配置请求失败", http.StatusBadRequest)
		return
	}

	if err := normalizeMCPConfig(req); err != nil {
		h.logger.Error("校验MCP配置失败", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.storage.MCP().Save(req)
	if err != nil {
		h.logger.Error("保存MCP配置失败", "error", err)
		http.Error(w, "保存MCP配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.MCPConfig]{
		Code:    http.StatusOK,
		Message: "MCP配置创建成功",
		Data:    req,
	})
}

func (h *MCPHandler) GetRuntimeAll(w http.ResponseWriter, r *http.Request) {
	configs, err := h.storage.MCP().List()
	if err != nil {
		h.logger.Error("获取MCP运行态失败", "error", err)
		http.Error(w, "获取MCP运行态失败", http.StatusInternalServerError)
		return
	}

	items := make([]MCPRuntimeInfo, 0, len(configs))
	for _, cfg := range configs {
		items = append(items, h.runtimeInfoFor(cfg))
	}

	models.WriteData(w, models.BaseResponse[[]MCPRuntimeInfo]{
		Code:    http.StatusOK,
		Message: "MCP运行态获取成功",
		Data:    items,
	})
}

func (h *MCPHandler) Connect(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定MCP连接请求失败", "error", err)
		http.Error(w, "绑定MCP连接请求失败", http.StatusBadRequest)
		return
	}

	cfg, err := h.storage.MCP().GetByID(id)
	if err != nil {
		h.logger.Error("获取MCP配置失败", "error", err)
		http.Error(w, "获取MCP配置失败", http.StatusInternalServerError)
		return
	}

	if err := normalizeMCPConfig(cfg); err != nil {
		h.logger.Error("校验MCP配置失败", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !cfg.Enabled {
		http.Error(w, "MCP 未启用，无法建立运行态连接", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), managedMCPTimeout(cfg))
	defer cancel()

	client, err := h.connectWithManager(ctx, cfg)
	if err != nil {
		h.logger.Error("连接MCP失败", "name", cfg.Name, "error", err)
		http.Error(w, "连接MCP失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	info := h.runtimeInfoFor(cfg)
	if len(info.Tools) == 0 {
		info.Tools = client.GetToolNames()
		info.ToolCount = len(info.Tools)
	}

	models.WriteData(w, models.BaseResponse[MCPRuntimeInfo]{
		Code:    http.StatusOK,
		Message: "MCP连接成功",
		Data:    info,
	})
}
