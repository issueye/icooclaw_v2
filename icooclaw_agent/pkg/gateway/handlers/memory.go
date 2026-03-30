package handlers

import (
	"log/slog"
	"net/http"

	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/storage"
)

type MemoryHandler struct {
	*crudResourceHandler[*storage.Memory, *storage.QueryMemory, *storage.ResQueryMemory, []*storage.Memory]
	logger  *slog.Logger
	storage *storage.Storage
}

func NewMemoryHandler(logger *slog.Logger, st *storage.Storage) *MemoryHandler {
	return &MemoryHandler{
		logger:  logger,
		storage: st,
		crudResourceHandler: &crudResourceHandler[*storage.Memory, *storage.QueryMemory, *storage.ResQueryMemory, []*storage.Memory]{
			logger: logger,
			messages: resourceMessages{
				BindPage:      "绑定分页请求失败",
				PageFailed:    "获取记忆列表失败",
				PageSuccess:   "记忆列表获取成功",
				BindCreate:    "绑定创建记忆请求失败",
				CreateFailed:  "保存记忆失败",
				CreateSuccess: "记忆创建成功",
				BindUpdate:    "绑定保存记忆请求失败",
				UpdateFailed:  "保存记忆失败",
				UpdateSuccess: "记忆更新成功",
				BindDelete:    "绑定删除记忆请求失败",
				DeleteFailed:  "删除记忆失败",
				DeleteSuccess: "记忆删除成功",
				BindGet:       "绑定获取记忆请求失败",
				GetFailed:     "获取记忆失败",
				GetSuccess:    "记忆获取成功",
			},
			page:   st.Memory().Page,
			create: st.Memory().Save,
			update: st.Memory().Save,
			delete: st.Memory().Delete,
			get: func(id string) ([]*storage.Memory, error) {
				return st.Memory().Get(id, 100)
			},
		},
	}
}

func (h *MemoryHandler) Search(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[struct {
		Query string `json:"query"`
	}](r)
	if err != nil {
		h.logger.Error("绑定搜索记忆请求失败", "error", err)
		http.Error(w, "绑定搜索记忆请求失败", http.StatusBadRequest)
		return
	}

	memories, err := h.storage.Memory().Page(&storage.QueryMemory{Query: req.Query})
	if err != nil {
		h.logger.Error("搜索记忆失败", "error", err)
		http.Error(w, "搜索记忆失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ResQueryMemory]{
		Code:    http.StatusOK,
		Message: "记忆搜索成功",
		Data:    memories,
	})
}
