package handlers

import (
	"log/slog"

	"icooclaw/pkg/storage"
)

type ToolHandler struct {
	*standardResourceHandler[*storage.Tool, *storage.QueryTool, *storage.ResQueryTool]
}

func NewToolHandler(logger *slog.Logger, st *storage.Storage) *ToolHandler {
	return &ToolHandler{
		standardResourceHandler: &standardResourceHandler[*storage.Tool, *storage.QueryTool, *storage.ResQueryTool]{
			logger: logger,
			messages: resourceMessages{
				BindPage:          "绑定分页请求失败",
				PageFailed:        "获取工具列表失败",
				PageSuccess:       "工具列表获取成功",
				BindCreate:        "绑定创建工具请求失败",
				CreateFailed:      "创建工具失败",
				CreateSuccess:     "工具创建成功",
				BindUpdate:        "绑定更新工具请求失败",
				UpdateFailed:      "更新工具失败",
				UpdateSuccess:     "工具更新成功",
				BindDelete:        "绑定删除工具请求失败",
				DeleteFailed:      "删除工具失败",
				DeleteSuccess:     "工具删除成功",
				BindGet:           "绑定获取工具请求失败",
				GetFailed:         "获取工具失败",
				GetSuccess:        "工具获取成功",
				GetAllFailed:      "获取所有工具失败",
				GetAllSuccess:     "工具列表获取成功",
				GetEnabledFailed:  "获取启用工具失败",
				GetEnabledSuccess: "启用工具列表获取成功",
			},
			page:        st.Tool().Page,
			create:      st.Tool().SaveTool,
			update:      st.Tool().SaveTool,
			delete:      st.Tool().DeleteTool,
			get:         st.Tool().GetTool,
			list:        st.Tool().ListTools,
			listEnabled: st.Tool().ListEnabledTools,
		},
	}
}
