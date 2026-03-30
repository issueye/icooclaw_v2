package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"icooclaw/pkg/agent"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/storage"
)

type WorkspaceHandler struct {
	logger       *slog.Logger
	storage      *storage.Storage
	agentManager *agent.AgentManager
}

func NewWorkspaceHandler(logger *slog.Logger, st *storage.Storage, agentManager *agent.AgentManager) *WorkspaceHandler {
	return &WorkspaceHandler{
		logger:       logger,
		storage:      st,
		agentManager: agentManager,
	}
}

type workspacePromptPayload struct {
	Name    string `json:"name"`
	Content string `json:"content,omitempty"`
}

type generateWorkspacePromptRequest struct {
	Name        string `json:"name"`
	Instruction string `json:"instruction"`
	Current     string `json:"current,omitempty"`
	Mode        string `json:"mode,omitempty"`
}

func (h *WorkspaceHandler) GetPrompt(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*workspacePromptPayload](r)
	if err != nil {
		http.Error(w, "绑定工作区提示词请求失败", http.StatusBadRequest)
		return
	}
	name, err := storage.NormalizeWorkspacePromptName(req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	content, err := h.storage.Workspace().Load(name)
	if err != nil {
		h.logger.Error("读取工作区提示词失败", "name", name, "error", err)
		http.Error(w, "读取工作区提示词失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[map[string]any]{
		Code:    http.StatusOK,
		Message: "工作区提示词读取成功",
		Data: map[string]any{
			"name":    name,
			"content": content,
		},
	})
}

func (h *WorkspaceHandler) SavePrompt(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*workspacePromptPayload](r)
	if err != nil {
		http.Error(w, "绑定工作区提示词保存请求失败", http.StatusBadRequest)
		return
	}
	name, err := storage.NormalizeWorkspacePromptName(req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.storage.Workspace().Save(name, req.Content); err != nil {
		h.logger.Error("保存工作区提示词失败", "name", name, "error", err)
		http.Error(w, "保存工作区提示词失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[map[string]any]{
		Code:    http.StatusOK,
		Message: "工作区提示词保存成功",
		Data: map[string]any{
			"name":    name,
			"content": req.Content,
		},
	})
}

func (h *WorkspaceHandler) GeneratePrompt(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*generateWorkspacePromptRequest](r)
	if err != nil {
		http.Error(w, "绑定工作区提示词生成请求失败", http.StatusBadRequest)
		return
	}
	if h.agentManager == nil {
		http.Error(w, "AI 生成功能未启用", http.StatusServiceUnavailable)
		return
	}

	name, err := storage.NormalizeWorkspacePromptName(req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	instruction := strings.TrimSpace(req.Instruction)
	if instruction == "" {
		http.Error(w, "instruction 不能为空", http.StatusBadRequest)
		return
	}

	prompt := buildWorkspacePromptInstruction(name, strings.TrimSpace(req.Mode), instruction, req.Current)
	sessionID := fmt.Sprintf("workspace-prompt-%s-%d", strings.ToLower(name), time.Now().UnixNano())
	result, err := h.agentManager.RunAgent(bus.InboundMessage{
		Channel:   consts.WEBSOCKET,
		SessionID: sessionID,
		Sender: bus.SenderInfo{
			ID:    "workspace-settings",
			Name:  "workspace-settings",
			IsBot: false,
		},
		Text:      prompt,
		Timestamp: time.Now(),
	})
	if err != nil {
		h.logger.Error("AI 生成工作区提示词失败", "name", name, "error", err)
		http.Error(w, "AI 生成工作区提示词失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[map[string]any]{
		Code:    http.StatusOK,
		Message: "工作区提示词生成成功",
		Data: map[string]any{
			"name":    name,
			"content": strings.TrimSpace(result),
		},
	})
}

func buildWorkspacePromptInstruction(name, mode, instruction, current string) string {
	action := "生成"
	if strings.EqualFold(mode, "modify") || strings.TrimSpace(current) != "" {
		action = "改写"
	}

	var targetDesc string
	switch name {
	case storage.WorkspacePromptSoul:
		targetDesc = "SOUL.md，定义 AI 的角色灵魂、气质、价值观与表达风格。"
	case storage.WorkspacePromptUser:
		targetDesc = "USER.md，定义用户画像、偏好、协作习惯和使用约束。"
	default:
		targetDesc = name + ".md。"
	}

	builder := strings.Builder{}
	builder.WriteString("你是一个提示词编辑器。请根据要求")
	builder.WriteString(action)
	builder.WriteString("工作区文件内容。\n")
	builder.WriteString("目标文件：")
	builder.WriteString(targetDesc)
	builder.WriteString("\n")
	builder.WriteString("要求：\n")
	builder.WriteString("1. 只返回最终 Markdown 内容，不要解释，不要加代码块。\n")
	builder.WriteString("2. 保持中文，表达清晰、精炼、可直接保存。\n")
	builder.WriteString("3. 使用合适的标题和结构，但不要过度冗长。\n")
	builder.WriteString("4. 如果是改写，保留原文中合理的信息并按要求优化。\n\n")
	builder.WriteString("用户要求：\n")
	builder.WriteString(instruction)
	builder.WriteString("\n")
	if strings.TrimSpace(current) != "" {
		builder.WriteString("\n当前内容：\n")
		builder.WriteString(current)
		builder.WriteString("\n")
	}

	return builder.String()
}
