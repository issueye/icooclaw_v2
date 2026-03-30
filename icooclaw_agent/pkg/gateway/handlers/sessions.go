package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/storage"
)

type SessionHandler struct {
	*crudResourceHandler[*storage.Session, *storage.QuerySession, *storage.ResQuerySession, *storage.Session]
	logger  *slog.Logger
	storage *storage.Storage
}

func NewSessionHandler(logger *slog.Logger, st *storage.Storage) *SessionHandler {
	return &SessionHandler{
		logger:  logger,
		storage: st,
		crudResourceHandler: &crudResourceHandler[*storage.Session, *storage.QuerySession, *storage.ResQuerySession, *storage.Session]{
			logger: logger,
			messages: resourceMessages{
				BindPage:      "绑定分页请求失败",
				PageFailed:    "获取会话列表失败",
				PageSuccess:   "会话列表获取成功",
				BindDelete:    "绑定获取消息请求失败",
				DeleteFailed:  "删除会话失败",
				DeleteSuccess: "会话删除成功",
				BindGet:       "绑定获取消息请求失败",
				GetFailed:     "获取会话失败",
				GetSuccess:    "会话获取成功",
			},
			page: func(req *storage.QuerySession) (*storage.ResQuerySession, error) {
				if req.Channel == "" {
					req.Channel = consts.WEBSOCKET
				}
				return st.Session().Page(req)
			},
			delete: st.Session().Delete,
			get:    st.Session().Get,
		},
	}
}

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	Channel   string            `json:"channel,omitempty"`
	UserID    string            `json:"user_id,omitempty"`
	SessionID string            `json:"session_id,omitempty"`
	AgentID   string            `json:"agent_id,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// CreateSessionResponse 创建会话响应
type CreateSessionResponse struct {
	SessionID string `json:"session_id"`
	Channel   string `json:"channel"`
	UserID    string `json:"user_id"`
	AgentID   string `json:"agent_id,omitempty"`
}

// Create 创建新会话 (供前端调用)
func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*CreateSessionRequest](r)
	if err != nil {
		h.logger.Error("绑定创建会话请求失败", "error", err)
		http.Error(w, "绑定创建会话请求失败", http.StatusBadRequest)
		return
	}

	h.logger.Info("创建会话", slog.Any("params", req))

	if req.Channel == "" {
		req.Channel = consts.WEBSOCKET
	}

	if req.SessionID == "" {
		req.SessionID = fmt.Sprintf("session-%d", time.Now().UnixNano())
	}

	session := storage.Session{
		Channel: req.Channel,
		UserID:  req.UserID,
		AgentID: req.AgentID,
		Title:   req.Metadata["title"],
	}

	if req.SessionID != "" {
		session.ID = req.SessionID
	}

	h.logger.Info("创建会话", slog.Any("params", session))

	err = h.storage.Session().Save(&session)
	if err != nil {
		h.logger.Error("创建会话失败", "error", err.Error())
		http.Error(w, "创建会话失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*CreateSessionResponse]{
		Code:    http.StatusOK,
		Message: "会话创建成功",
		Data: &CreateSessionResponse{
			SessionID: session.ID,
			Channel:   session.Channel,
			UserID:    session.UserID,
			AgentID:   session.AgentID,
		},
	})
}

// Save 保存会话
func (h *SessionHandler) Save(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Session](r)
	if err != nil {
		h.logger.Error("绑定保存会话请求失败", "error", err)
		http.Error(w, "绑定保存会话请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Session().Save(req)
	if err != nil {
		h.logger.Error("保存会话失败", "error", err)
		http.Error(w, "保存会话失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Session]{
		Code:    http.StatusOK,
		Message: "会话保存成功",
		Data:    req,
	})
}
