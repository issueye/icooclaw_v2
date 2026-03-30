package handlers

import (
	"log/slog"
	"net/http"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/storage"
)

type MessageHandler struct {
	*crudResourceHandler[*storage.Message, *QueryMessageRequest, *storage.ResQueryMessage, *storage.Message]
	logger  *slog.Logger
	storage *storage.Storage
}

func NewMessageHandler(logger *slog.Logger, st *storage.Storage) *MessageHandler {
	return &MessageHandler{
		logger:  logger,
		storage: st,
		crudResourceHandler: &crudResourceHandler[*storage.Message, *QueryMessageRequest, *storage.ResQueryMessage, *storage.Message]{
			logger: logger,
			messages: resourceMessages{
				BindPage:      "绑定分页请求失败",
				PageFailed:    "获取消息列表失败",
				PageSuccess:   "消息列表获取成功",
				BindCreate:    "绑定创建消息请求失败",
				CreateFailed:  "创建消息失败",
				CreateSuccess: "消息创建成功",
				BindUpdate:    "绑定更新消息请求失败",
				UpdateFailed:  "更新消息失败",
				UpdateSuccess: "消息更新成功",
				BindDelete:    "绑定删除消息请求失败",
				DeleteFailed:  "删除消息失败",
				DeleteSuccess: "消息删除成功",
				BindGet:       "绑定获取消息请求失败",
				GetFailed:     "获取消息失败",
				GetSuccess:    "消息获取成功",
			},
			page: func(req *QueryMessageRequest) (*storage.ResQueryMessage, error) {
				query := &storage.QueryMessage{
					Page:    req.Page,
					Role:    req.Role,
					KeyWord: req.KeyWord,
				}
				if req.SessionID != "" {
					channel := req.Channel
					if channel == "" {
						channel = consts.WEBSOCKET
					}
					query.SessionID = consts.GetSessionKey(channel, req.SessionID)
				}
				return st.Message().Page(query)
			},
			create: st.Message().Save,
			update: st.Message().Save,
			delete: st.Message().Delete,
			get:    st.Message().GetByID,
		},
	}
}

// QueryMessageRequest 消息查询请求
type QueryMessageRequest struct {
	Page      storage.Page `json:"page"`
	SessionID string       `json:"session_id"`
	Channel   string       `json:"channel"`
	Role      string       `json:"role"`
	KeyWord   string       `json:"key_word"`
}

// Save 保存消息
func (h *MessageHandler) Save(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Message](r)
	if err != nil {
		h.logger.Error("绑定保存消息请求失败", "error", err)
		http.Error(w, "绑定保存消息请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Message().Save(req)
	if err != nil {
		h.logger.Error("保存消息失败", "error", err)
		http.Error(w, "保存消息失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Message]{
		Code:    http.StatusOK,
		Message: "消息保存成功",
		Data:    req,
	})
}

func (h *MessageHandler) GetBySessionID(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*GetBySessionIDRequest](r)
	if err != nil {
		h.logger.Error("绑定获取消息请求失败", "error", err)
		http.Error(w, "绑定获取消息请求失败", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		h.logger.Error("会话ID不能为空")
		http.Error(w, "会话ID不能为空", http.StatusBadRequest)
		return
	}

	if req.Channel == "" {
		req.Channel = consts.WEBSOCKET
	}

	sessionKey := consts.GetSessionKey(req.Channel, req.SessionID)

	messages, err := h.storage.Message().Get(sessionKey, 100)
	if err != nil {
		h.logger.Error("获取消息失败", "error", err)
		http.Error(w, "获取消息失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.Message]{
		Code:    http.StatusOK,
		Message: "消息获取成功",
		Data:    messages,
	})
}

// GetBySessionIDRequest 按会话ID获取消息请求
type GetBySessionIDRequest struct {
	Channel   string `json:"channel"`
	SessionID string `json:"session_id"`
}
