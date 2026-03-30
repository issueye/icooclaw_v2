package handlers

import (
	"fmt"
	"log/slog"
	"strings"

	"icooclaw/pkg/storage"
)

type AgentHandler struct {
	*standardResourceHandler[*storage.Agent, *storage.QueryAgent, *storage.ResQueryAgent]
}

func NewAgentHandler(logger *slog.Logger, st *storage.Storage) *AgentHandler {
	normalizeAgent := func(agent *storage.Agent, requireID bool) error {
		if requireID && strings.TrimSpace(agent.ID) == "" {
			return fmt.Errorf("智能体ID不能为空")
		}
		if !storage.IsValidAgentType(agent.Type) {
			return fmt.Errorf("智能体类型无效，仅支持 master 或 subagent")
		}
		agent.Type = storage.NormalizeAgentType(agent.Type)
		return nil
	}

	return &AgentHandler{
		standardResourceHandler: &standardResourceHandler[*storage.Agent, *storage.QueryAgent, *storage.ResQueryAgent]{
			logger: logger,
			messages: resourceMessages{
				BindPage:          "绑定分页请求失败",
				PageFailed:        "获取智能体列表失败",
				PageSuccess:       "智能体列表获取成功",
				BindCreate:        "绑定创建智能体请求失败",
				CreateFailed:      "创建智能体失败",
				CreateSuccess:     "智能体创建成功",
				BindUpdate:        "绑定更新智能体请求失败",
				UpdateFailed:      "更新智能体失败",
				UpdateSuccess:     "智能体更新成功",
				BindDelete:        "绑定删除智能体请求失败",
				DeleteFailed:      "删除智能体失败",
				DeleteSuccess:     "智能体删除成功",
				BindGet:           "绑定获取智能体请求失败",
				GetFailed:         "获取智能体失败",
				GetSuccess:        "智能体获取成功",
				GetAllFailed:      "获取所有智能体失败",
				GetAllSuccess:     "智能体列表获取成功",
				GetEnabledFailed:  "获取启用智能体失败",
				GetEnabledSuccess: "启用智能体列表获取成功",
			},
			page:         st.Agent().Page,
			create:       st.Agent().Save,
			update:       st.Agent().Save,
			delete:       st.Agent().Delete,
			get:          st.Agent().GetByID,
			list:         st.Agent().List,
			listEnabled:  st.Agent().ListEnabled,
			beforeCreate: func(agent *storage.Agent) error { return normalizeAgent(agent, false) },
			beforeUpdate: func(agent *storage.Agent) error { return normalizeAgent(agent, true) },
		},
	}
}
