package handlers

import (
	"log/slog"

	"icooclaw/pkg/storage"
)

type ProviderHandler struct {
	*standardResourceHandler[*storage.Provider, *storage.QueryProvider, *storage.ResQueryProvider]
}

func NewProviderHandler(logger *slog.Logger, st *storage.Storage) *ProviderHandler {
	return &ProviderHandler{
		standardResourceHandler: &standardResourceHandler[*storage.Provider, *storage.QueryProvider, *storage.ResQueryProvider]{
			logger: logger,
			messages: resourceMessages{
				BindPage:          "绑定分页请求失败",
				PageFailed:        "获取Provider配置列表失败",
				PageSuccess:       "Provider配置列表获取成功",
				BindCreate:        "绑定创建供应商配置请求失败",
				CreateFailed:      "创建供应商配置失败",
				CreateSuccess:     "Provider配置创建成功",
				BindUpdate:        "绑定更新供应商配置请求失败",
				UpdateFailed:      "更新供应商配置失败",
				UpdateSuccess:     "供应商配置更新成功",
				BindDelete:        "绑定删除Provider配置请求失败",
				DeleteFailed:      "删除Provider配置失败",
				DeleteSuccess:     "Provider配置删除成功",
				BindGet:           "绑定获取Provider配置请求失败",
				GetFailed:         "获取Provider配置失败",
				GetSuccess:        "Provider配置获取成功",
				GetAllFailed:      "获取所有供应商配置失败",
				GetAllSuccess:     "供应商配置列表获取成功",
				GetEnabledFailed:  "获取启用Provider配置失败",
				GetEnabledSuccess: "启用Provider配置列表获取成功",
			},
			page:        st.Provider().Page,
			create:      st.Provider().Save,
			update:      st.Provider().Save,
			delete:      st.Provider().Delete,
			get:         st.Provider().GetByName,
			list:        st.Provider().List,
			listEnabled: st.Provider().List,
		},
	}
}
