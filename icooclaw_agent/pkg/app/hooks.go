package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"icooclaw/pkg/agent/react"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/scheduler"
	"icooclaw/pkg/script"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
	"log/slog"
)

// AppAgentHooks 应用智能体钩子实现
// 支持通过 JavaScript 脚本扩展钩子功能
type AppAgentHooks struct {
	logger          *slog.Logger
	workspace       string
	hooksDir        string
	scriptEngine    *script.Engine
	hookScripts     map[string]hookScript
	mu              sync.Mutex
	storage         *storage.Storage
	scheduler       *scheduler.Scheduler
	providerManager *providers.Manager
	summaryModel    string
	summaryAgent    *react.ReActAgent
	keepRecent      int
	summaryBatch    int
}

type HooksDependencies struct {
	Logger       *slog.Logger
	Workspace    string
	Storage      *storage.Storage
	Scheduler    *scheduler.Scheduler
	SummaryAgent *react.ReActAgent
	RecentCount  int
}

// NewAppAgentHooks 创建带脚本引擎的应用智能体钩子
func NewAppAgentHooks(deps HooksDependencies) *AppAgentHooks {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	if deps.RecentCount <= 0 {
		deps.RecentCount = consts.DefaultRecentCount
	}

	summaryBatch := deps.RecentCount * 2
	if summaryBatch < consts.DefaultHookSummaryMinBatch {
		summaryBatch = consts.DefaultHookSummaryMinBatch
	}

	hooks := &AppAgentHooks{
		logger:       deps.Logger,
		storage:      deps.Storage,
		scheduler:    deps.Scheduler,
		summaryAgent: deps.SummaryAgent,
		hookScripts:  make(map[string]hookScript),
		hooksDir:     consts.HookScriptDir,
		keepRecent:   deps.RecentCount,
		summaryBatch: summaryBatch,
	}

	hooks.SetWorkspace(deps.Workspace)
	return hooks
}

// OnGetProvider 获取供应商实例
func (h *AppAgentHooks) OnGetProvider(ctx context.Context, defaultModel string, storage *storage.ProviderStorage) error {
	h.logger.Info("获取供应商", "default_model", defaultModel)

	result, err := h.executeHook(ctx, consts.HookScriptDir, consts.HookGetProvider, defaultModel)
	if err != nil {
		h.logger.Warn("执行 onGetProvider 钩子失败", "error", err)
	}

	if result != nil {
		if model, ok := result.(string); ok && model != "" {
			h.logger.Info("使用 JS 钩子返回的模型", "model", model)
			_ = model
		}
	}

	return nil
}

// OnCreateAgent 创建智能体实例
func (h *AppAgentHooks) OnCreateAgent(ctx context.Context, a *react.ReActAgent) (*react.ReActAgent, error) {
	h.logger.Info("创建智能体", "agent", a)

	result, err := h.executeHook(ctx, consts.HookScriptDir, consts.HookCreateAgent, mustMarshalJSON(a))
	if err != nil {
		h.logger.Warn("执行 onCreateAgent 钩子失败", "error", err)
	}

	if result != nil {
		h.logger.Info("使用 JS 钩子返回的智能体配置", "result", result)
	}

	return a, nil
}

// OnBuildMessagesBefore 构建消息列表前钩子
func (h *AppAgentHooks) OnBuildMessagesBefore(ctx context.Context, sessionKey string, msg bus.InboundMessage, history []providers.ChatMessage) ([]providers.ChatMessage, error) {
	result, err := h.executeHook(ctx, consts.HookScriptDir, consts.HookBuildMessagesBefore, sessionKey, mustMarshalJSON(msg), mustMarshalJSON(history))
	if err != nil {
		h.logger.Warn("执行 onBuildMessagesBefore 钩子失败", "error", err)
		return history, nil
	}

	if messages, ok, err := decodeHookMessages(result); err == nil && ok {
		return messages, nil
	}

	return history, nil
}

// OnBuildMessagesAfter 构建消息列表后钩子
func (h *AppAgentHooks) OnBuildMessagesAfter(ctx context.Context, sessionKey string, msg bus.InboundMessage, history []providers.ChatMessage) ([]providers.ChatMessage, error) {
	result, err := h.executeHook(ctx, consts.HookScriptDir, consts.HookBuildMessagesAfter, sessionKey, mustMarshalJSON(msg), mustMarshalJSON(history))
	if err != nil {
		h.logger.Warn("执行 onBuildMessagesAfter 钩子失败", "error", err)
		return history, nil
	}

	if messages, ok, err := decodeHookMessages(result); err == nil && ok {
		return messages, nil
	}

	return history, nil
}

// OnRunLLMBefore 运行LLM模型前钩子
func (h *AppAgentHooks) OnRunLLMBefore(ctx context.Context, msg bus.InboundMessage, history []providers.ChatMessage) ([]providers.ChatMessage, error) {
	result, err := h.executeHook(ctx, consts.HookScriptDir, consts.HookRunLLMBefore, mustMarshalJSON(msg), mustMarshalJSON(history))
	if err != nil {
		h.logger.Warn("执行 onRunLLMBefore 钩子失败", "error", err)
		return history, nil
	}

	if messages, ok, err := decodeHookMessages(result); err == nil && ok {
		return messages, nil
	}

	return history, nil
}

// OnRunLLMAfter 运行LLM模型后钩子
func (h *AppAgentHooks) OnRunLLMAfter(ctx context.Context, msg bus.InboundMessage, history []providers.ChatMessage) ([]providers.ChatMessage, error) {
	result, err := h.executeHook(ctx, consts.HookScriptDir, consts.HookRunLLMAfter, mustMarshalJSON(msg), mustMarshalJSON(history))
	if err != nil {
		h.logger.Warn("执行 onRunLLMAfter 钩子失败", "error", err)
		return history, nil
	}

	if messages, ok, err := decodeHookMessages(result); err == nil && ok {
		return messages, nil
	}

	return history, nil
}

// OnToolCallBefore 工具调用前钩子
func (h *AppAgentHooks) OnToolCallBefore(ctx context.Context, toolName string, tc providers.ToolCall, msg bus.InboundMessage) (providers.ToolCall, error) {
	result, err := h.executeHook(ctx, consts.HookScriptDir, consts.HookToolCallBefore, toolName, mustMarshalJSON(tc), mustMarshalJSON(msg))
	if err != nil {
		h.logger.Warn("执行 onToolCallBefore 钩子失败", "error", err)
		return tc, nil
	}

	if updated, ok, err := decodeHookToolCall(result); err == nil && ok {
		return updated, nil
	}

	return tc, nil
}

// OnToolCallAfter 工具调用后钩子
func (h *AppAgentHooks) OnToolCallAfter(ctx context.Context, toolName string, msg bus.InboundMessage, result *tools.Result) error {
	_, err := h.executeHook(ctx, consts.HookScriptDir, consts.HookToolCallAfter, toolName, mustMarshalJSON(msg), mustMarshalJSON(result))
	if err != nil {
		h.logger.Warn("执行 onToolCallAfter 钩子失败", "error", err)
	}

	return nil
}

// OnToolParseArguments 工具参数解析钩子
func (h *AppAgentHooks) OnToolParseArguments(ctx context.Context, toolName string, tc providers.ToolCall, msg bus.InboundMessage) (map[string]any, error) {
	result, err := h.executeHook(ctx, consts.HookScriptDir, consts.HookToolParseArguments, toolName, mustMarshalJSON(tc), mustMarshalJSON(msg))
	if err != nil {
		h.logger.Warn("执行 onToolParseArguments 钩子失败", "error", err)
	}

	var args map[string]any
	if tc.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, fmt.Errorf("解析参数失败: %w", err)
		}
	}

	if updatedArgs, ok, err := decodeHookArgs(result); err == nil && ok {
		return updatedArgs, nil
	}

	return args, nil
}

// OnAgentStart 智能体开始处理消息钩子
func (h *AppAgentHooks) OnAgentStart(ctx context.Context, msg bus.InboundMessage) error {
	h.logger.Info("智能体开始处理", "session_id", msg.SessionID, "channel", msg.Channel)

	_, err := h.executeHook(ctx, consts.HookScriptDir, consts.HookAgentStart, mustMarshalJSON(msg))
	if err != nil {
		h.logger.Warn("执行 onAgentStart 钩子失败", "error", err)
	}

	return nil
}

// OnAgentEnd 智能体结束处理消息钩子
func (h *AppAgentHooks) OnAgentEnd(ctx context.Context, msg bus.InboundMessage, response string, iteration int, messageID string) error {
	h.logger.Info("智能体结束处理", "session_id", msg.SessionID, "iteration", iteration, "response_len", len(response))

	msgJSON := mustMarshalJSON(msg)
	_, err := h.executeHook(ctx, consts.HookScriptDir, consts.HookAgentEnd, msgJSON, response, iteration)
	if err != nil {
		h.logger.Warn("执行 onAgentEnd 钩子失败", "error", err)
	}

	if h.summaryAgent != nil {
		sessionKey := consts.GetSessionKey(msg.Channel, msg.SessionID)
		summary, err := h.maybeSummarizeSession(ctx, msg, sessionKey, response, iteration)
		if err != nil {
			h.logger.Warn("更新会话摘要失败", "error", err, "session_key", sessionKey, "message_id", messageID)
			return err
		}
		if summary != "" {
			if _, err := h.executeHook(ctx, consts.HookScriptDir, consts.HookAgentEndWithSummary, msgJSON, response, iteration, summary); err != nil {
				h.logger.Warn("执行 onAgentEndWithSummary 钩子失败", "error", err)
			}
		}
	}

	return nil
}
