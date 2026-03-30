package handlers

import (
	"log/slog"

	"icooclaw/pkg/storage"
)

type ChannelHandler struct {
	*standardResourceHandler[*storage.Channel, *storage.QueryChannel, *storage.ResQueryChannel]
}

func NewChannelHandler(logger *slog.Logger, st *storage.Storage) *ChannelHandler {
	return &ChannelHandler{
		standardResourceHandler: &standardResourceHandler[*storage.Channel, *storage.QueryChannel, *storage.ResQueryChannel]{
			logger: logger,
			messages: resourceMessages{
				BindPage:          "绑定分页请求失败",
				PageFailed:        "获取通道配置列表失败",
				PageSuccess:       "通道配置列表获取成功",
				BindCreate:        "绑定创建通道配置请求失败",
				CreateFailed:      "创建通道配置失败",
				CreateSuccess:     "通道配置创建成功",
				BindUpdate:        "绑定更新通道配置请求失败",
				UpdateFailed:      "更新通道配置失败",
				UpdateSuccess:     "通道配置更新成功",
				BindDelete:        "绑定删除通道配置请求失败",
				DeleteFailed:      "删除通道配置失败",
				DeleteSuccess:     "通道配置删除成功",
				BindGet:           "绑定获取通道配置请求失败",
				GetFailed:         "获取通道配置失败",
				GetSuccess:        "通道配置获取成功",
				GetAllFailed:      "获取所有通道配置失败",
				GetAllSuccess:     "通道配置列表获取成功",
				GetEnabledFailed:  "获取启用通道配置失败",
				GetEnabledSuccess: "启用通道配置列表获取成功",
			},
			page:        st.Channel().Page,
			create:      st.Channel().SaveChannel,
			update:      st.Channel().SaveChannel,
			delete:      st.Channel().Delete,
			get:         st.Channel().GetChannel,
			list:        st.Channel().ListChannels,
			listEnabled: st.Channel().ListEnabledChannels,
		},
	}
}
