package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/envmgr"
	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/storage"
)

type ParamHandler struct {
	*standardResourceHandler[*storage.ParamConfig, *storage.QueryParam, *storage.ResQueryParam]
	logger  *slog.Logger
	storage *storage.Storage
}

func NewParamHandler(logger *slog.Logger, st *storage.Storage) *ParamHandler {
	return &ParamHandler{
		logger:  logger,
		storage: st,
		standardResourceHandler: &standardResourceHandler[*storage.ParamConfig, *storage.QueryParam, *storage.ResQueryParam]{
			logger: logger,
			messages: resourceMessages{
				BindPage:      "绑定分页请求失败",
				PageFailed:    "获取参数配置列表失败",
				PageSuccess:   "参数配置列表获取成功",
				BindCreate:    "绑定创建参数配置请求失败",
				CreateFailed:  "创建参数配置失败",
				CreateSuccess: "参数配置创建成功",
				BindUpdate:    "绑定更新参数配置请求失败",
				UpdateFailed:  "更新参数配置失败",
				UpdateSuccess: "参数配置更新成功",
				BindDelete:    "绑定删除参数配置请求失败",
				DeleteFailed:  "删除参数配置失败",
				DeleteSuccess: "参数配置删除成功",
				BindGet:       "绑定获取参数配置请求失败",
				GetFailed:     "获取参数配置失败",
				GetSuccess:    "参数配置获取成功",
				GetAllFailed:  "获取所有参数配置失败",
				GetAllSuccess: "参数配置列表获取成功",
			},
			page:   st.Param().Page,
			create: st.Param().Save,
			update: st.Param().Save,
			delete: st.Param().Delete,
			get:    st.Param().Get,
			list:   st.Param().List,
		},
	}
}

func (h *ParamHandler) GetByKey(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[struct {
		Key string `json:"key"`
	}](r)
	if err != nil {
		h.logger.Error("绑定获取参数配置请求失败", "error", err)
		http.Error(w, "绑定获取参数配置请求失败", http.StatusBadRequest)
		return
	}

	config, err := h.storage.Param().Get(req.Key)
	if err != nil {
		h.logger.Error("获取参数配置失败", "error", err)
		http.Error(w, "获取参数配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ParamConfig]{
		Code:    http.StatusOK,
		Message: "参数配置获取成功",
		Data:    config,
	})
}

func (h *ParamHandler) GetByGroup(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[struct {
		Group string `json:"group"`
	}](r)
	if err != nil {
		h.logger.Error("绑定获取参数配置请求失败", "error", err)
		http.Error(w, "绑定获取参数配置请求失败", http.StatusBadRequest)
		return
	}

	configs, err := h.storage.Param().ListByGroup(req.Group)
	if err != nil {
		h.logger.Error("获取参数配置失败", "error", err)
		http.Error(w, "获取参数配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.ParamConfig]{
		Code:    http.StatusOK,
		Message: "参数配置列表获取成功",
		Data:    configs,
	})
}

func (h *ParamHandler) SetDefaultModel(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[struct {
		Model string `json:"model"`
	}](r)
	if err != nil {
		h.logger.Error("绑定设置默认模型请求失败", "error", err)
		http.Error(w, "绑定设置默认模型请求失败", http.StatusBadRequest)
		return
	}

	if req.Model == "" {
		http.Error(w, "模型不能为空", http.StatusBadRequest)
		return
	}

	param := &storage.ParamConfig{
		Key:         consts.DEFAULT_MODEL_KEY,
		Value:       req.Model,
		Description: "AI Agent 默认使用的模型",
		Group:       "agent",
		Enabled:     true,
	}

	err = h.storage.Param().SaveOrUpdateByKey(param)
	if err != nil {
		h.logger.Error("设置默认模型失败", "error", err)
		http.Error(w, "设置默认模型失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "默认模型设置成功",
		Data: map[string]interface{}{
			"model": req.Model,
		},
	})
}

func (h *ParamHandler) GetDefaultModel(w http.ResponseWriter, r *http.Request) {
	config, err := h.storage.Param().Get(consts.DEFAULT_MODEL_KEY)
	if err != nil || config == nil {
		models.WriteData(w, models.BaseResponse[any]{
			Code:    http.StatusOK,
			Message: "未设置默认模型",
			Data:    nil,
		})
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "默认模型获取成功",
		Data: map[string]interface{}{
			"model": config.Value,
		},
	})
}

func (h *ParamHandler) SetDefaultAgent(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[struct {
		AgentID string `json:"agent_id"`
	}](r)
	if err != nil {
		h.logger.Error("绑定设置默认智能体请求失败", "error", err)
		http.Error(w, "绑定设置默认智能体请求失败", http.StatusBadRequest)
		return
	}

	agentID := strings.TrimSpace(req.AgentID)
	if agentID == "" {
		http.Error(w, "agent_id 不能为空", http.StatusBadRequest)
		return
	}
	agentInfo, err := h.storage.Agent().GetByID(agentID)
	if err != nil || agentInfo == nil {
		http.Error(w, "智能体不存在", http.StatusBadRequest)
		return
	}
	if agentInfo.Type != storage.AgentTypeMaster {
		http.Error(w, "默认智能体必须是 master 类型", http.StatusBadRequest)
		return
	}

	param := &storage.ParamConfig{
		Key:         consts.DEFAULT_AGENT_ID_KEY,
		Value:       agentInfo.ID,
		Description: "AI Agent 默认使用的智能体ID",
		Group:       "agent",
		Enabled:     true,
	}
	if err := h.storage.Param().SaveOrUpdateByKey(param); err != nil {
		h.logger.Error("设置默认智能体失败", "error", err)
		http.Error(w, "设置默认智能体失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "默认智能体设置成功",
		Data: map[string]any{
			"agent_id": agentInfo.ID,
			"name":     agentInfo.Name,
		},
	})
}

func (h *ParamHandler) GetDefaultAgent(w http.ResponseWriter, r *http.Request) {
	agentInfo, err := h.storage.ResolveDefaultAgent()
	if err != nil || agentInfo == nil {
		models.WriteData(w, models.BaseResponse[any]{
			Code:    http.StatusOK,
			Message: "未设置默认智能体",
			Data:    nil,
		})
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "默认智能体获取成功",
		Data: map[string]any{
			"agent_id":      agentInfo.ID,
			"name":          agentInfo.Name,
			"type":          agentInfo.Type,
			"description":   agentInfo.Description,
			"system_prompt": agentInfo.SystemPrompt,
		},
	})
}

func (h *ParamHandler) SetExecEnv(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[struct {
		Env map[string]string `json:"env"`
	}](r)
	if err != nil {
		h.logger.Error("绑定设置执行环境变量请求失败", "error", err)
		http.Error(w, "绑定设置执行环境变量请求失败", http.StatusBadRequest)
		return
	}

	if req.Env == nil {
		req.Env = map[string]string{}
	}
	if _, err := json.Marshal(req.Env); err != nil {
		http.Error(w, "环境变量必须是 JSON 对象", http.StatusBadRequest)
		return
	}

	if err := h.storage.ExecEnv().ReplaceAll(req.Env); err != nil {
		h.logger.Error("设置执行环境变量失败", "error", err)
		http.Error(w, "设置执行环境变量失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "执行环境变量设置成功",
		Data: map[string]any{
			"env": req.Env,
		},
	})
}

func (h *ParamHandler) GetExecEnv(w http.ResponseWriter, r *http.Request) {
	values, err := h.storage.ExecEnv().ToMap()
	if err != nil {
		h.logger.Error("获取执行环境变量失败", "error", err)
		http.Error(w, "获取执行环境变量失败", http.StatusInternalServerError)
		return
	}
	if len(values) == 0 && h.storage.Param() != nil {
		config, err := h.storage.Param().Get(consts.EXEC_ENV_KEY)
		if err != nil {
			h.logger.Error("获取旧版执行环境变量失败", "error", err)
			http.Error(w, "获取执行环境变量失败", http.StatusInternalServerError)
			return
		}
		if config != nil && config.Enabled {
			values, err = envmgr.ParseJSON(config.Value)
			if err != nil {
				h.logger.Error("解析旧版执行环境变量失败", "error", err)
				http.Error(w, "解析执行环境变量失败", http.StatusInternalServerError)
				return
			}
		}
	}
	if len(values) == 0 {
		models.WriteData(w, models.BaseResponse[any]{
			Code:    http.StatusOK,
			Message: "未设置执行环境变量",
			Data: map[string]any{
				"env": map[string]string{},
			},
		})
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "执行环境变量获取成功",
		Data: map[string]any{
			"env": values,
		},
	})
}
