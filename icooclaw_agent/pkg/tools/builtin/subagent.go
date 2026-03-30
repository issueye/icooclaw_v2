package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"icooclaw/pkg/agent/react"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/scheduler"
	"icooclaw/pkg/skill"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"

	"github.com/google/uuid"
)

const (
	subAgentToolName          = "run_subagent"
	defaultSubAgentIterations = 4
)

type subAgentTaskPayload struct {
	Prompt          string `json:"prompt"`
	ParentChannel   string `json:"parent_channel"`
	ParentSessionID string `json:"parent_session_id"`
	ChildSessionID  string `json:"child_session_id"`
	AgentID         string `json:"agent_id,omitempty"`
	Mode            string `json:"mode,omitempty"`
}

// SubAgentTool delegates a focused task to a background subagent task.
type SubAgentTool struct {
	storage         *storage.Storage
	scheduler       *scheduler.Scheduler
	providerManager *providers.Manager
	skills          skill.Loader
	registry        *tools.Registry
	logger          *slog.Logger
}

func NewSubAgentTool(
	st *storage.Storage,
	taskScheduler *scheduler.Scheduler,
	providerManager *providers.Manager,
	skills skill.Loader,
	registry *tools.Registry,
	logger *slog.Logger,
) *SubAgentTool {
	if logger == nil {
		logger = slog.Default()
	}
	return &SubAgentTool{
		storage:         st,
		scheduler:       taskScheduler,
		providerManager: providerManager,
		skills:          skills,
		registry:        registry,
		logger:          logger,
	}
}

func (t *SubAgentTool) Name() string {
	return subAgentToolName
}

func (t *SubAgentTool) Description() string {
	return "把一个聚焦的子任务委托给后台 subagent 任务处理，并返回任务编号。适合拆分复杂问题、局部研究或独立执行的小任务。"
}

func (t *SubAgentTool) Parameters() map[string]any {
	return map[string]any{
		"task": map[string]any{
			"type":        "string",
			"description": "subagent 要完成的具体任务，必须自包含且边界清晰。",
		},
		"context": map[string]any{
			"type":        "string",
			"description": "可选补充上下文，提供给 subagent 的额外约束、背景或输入。",
		},
		"agent_id": map[string]any{
			"type":        "string",
			"description": "可选 agent ID。提供时 subagent 使用对应 agent；不提供则使用默认 agent。",
		},
		"mode": map[string]any{
			"type":        "string",
			"description": "执行模式：async 异步投递后立即返回；sync 同步等待 subagent 完成并返回结果。默认 async。",
			"enum":        []string{"async", "sync"},
		},
		"timeout_seconds": map[string]any{
			"type":        "integer",
			"description": "sync 模式下等待结果的超时时间，默认 120 秒。",
		},
	}
}

func (t *SubAgentTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	if t.storage == nil || t.providerManager == nil || t.registry == nil || t.scheduler == nil {
		return &tools.Result{
			Success: false,
			Error:   fmt.Errorf("subagent tool is not configured"),
		}
	}

	taskContent := strings.TrimSpace(stringArg(args, "task"))
	if taskContent == "" {
		return &tools.Result{
			Success: false,
			Error:   fmt.Errorf("task is required"),
		}
	}

	prompt := taskContent
	if extra := strings.TrimSpace(stringArg(args, "context")); extra != "" {
		prompt = fmt.Sprintf("任务:\n%s\n\n补充上下文:\n%s", taskContent, extra)
	}
	mode := normalizeSubAgentMode(stringArg(args, "mode"))
	timeout := intArg(args, "timeout_seconds", 120)
	if timeout <= 0 {
		timeout = 120
	}

	toolCtx := tools.GetToolContext(ctx)
	channel := ""
	parentSessionID := ""
	if toolCtx != nil {
		channel = toolCtx.Channel
		parentSessionID = toolCtx.SessionID
	}

	payload, err := json.Marshal(subAgentTaskPayload{
		Prompt:          prompt,
		ParentChannel:   channel,
		ParentSessionID: parentSessionID,
		ChildSessionID:  buildSubAgentSessionID(parentSessionID),
		AgentID:         strings.TrimSpace(stringArg(args, "agent_id")),
		Mode:            mode,
	})
	if err != nil {
		return &tools.Result{
			Success: false,
			Error:   err,
		}
	}

	task := &storage.Task{
		Name:        fmt.Sprintf("subagent-%s", uuid.NewString()),
		Description: "后台 subagent 委托任务",
		TaskType:    scheduler.TaskTypeImmediate,
		Executor:    scheduler.TaskExecutorSubAgent,
		Params:      string(payload),
		Enabled:     true,
		LastStatus:  scheduler.TaskStatusPending,
	}
	if err := t.storage.Task().Create(task); err != nil {
		return &tools.Result{
			Success: false,
			Error:   fmt.Errorf("create subagent task failed: %w", err),
		}
	}
	resultCh, err := t.scheduler.ApplyStorageTask(task)
	if err != nil {
		return &tools.Result{
			Success: false,
			Error:   fmt.Errorf("dispatch subagent task failed: %w", err),
		}
	}
	if mode == "sync" {
		return t.waitSubAgentResult(ctx, resultCh, time.Duration(timeout)*time.Second)
	}

	return &tools.Result{
		Success: true,
		Content: fmt.Sprintf("Subagent task queued.\nTask ID: %s\nChild Session: %s", task.ID, buildTaskChildSessionID(payload)),
	}
}

func (t *SubAgentTool) waitSubAgentResult(ctx context.Context, resultCh <-chan scheduler.TaskResult, timeout time.Duration) *tools.Result {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case <-waitCtx.Done():
		return &tools.Result{
			Success: false,
			Error:   fmt.Errorf("wait subagent result timeout after %s", timeout),
		}
	case result, ok := <-resultCh:
		if !ok {
			return &tools.Result{
				Success: false,
				Error:   fmt.Errorf("subagent result channel closed"),
			}
		}
		if result.Error != nil {
			return &tools.Result{Success: false, Error: result.Error}
		}
		return &tools.Result{Success: true, Content: strings.TrimSpace(result.Result)}
	}
}

func NewSubAgentTaskExecutor(
	st *storage.Storage,
	providerManager *providers.Manager,
	skills skill.Loader,
	registry *tools.Registry,
	messageBus *bus.MessageBus,
	logger *slog.Logger,
) scheduler.TaskExecutor {
	if logger == nil {
		logger = slog.Default()
	}

	return func(ctx context.Context, task *scheduler.Task) (string, error) {
		if st == nil || providerManager == nil || registry == nil {
			return "", fmt.Errorf("subagent task executor is not configured")
		}

		var payload subAgentTaskPayload
		if err := json.Unmarshal([]byte(task.Params), &payload); err != nil {
			return "", fmt.Errorf("解析 subagent 任务参数失败: %w", err)
		}

		childRegistry := registry.CloneWithout(subAgentToolName)
		childAgent, err := react.NewReActAgentNoHooks(ctx, react.Dependencies{
			Tools:             childRegistry,
			Skills:            skills,
			Storage:           st,
			ProviderManager:   providerManager,
			Logger:            logger,
			MaxToolIterations: defaultSubAgentIterations,
		})
		if err != nil {
			return "", fmt.Errorf("create subagent failed: %w", err)
		}

		start := time.Now()
		content, iteration, err := childAgent.Chat(ctx, bus.InboundMessage{
			Channel:   payload.ParentChannel,
			SessionID: payload.ChildSessionID,
			Sender: bus.SenderInfo{
				ID:    "subagent",
				Name:  "subagent",
				IsBot: true,
			},
			Text:      payload.Prompt,
			Timestamp: start,
			Metadata: map[string]any{
				"parent_session_id": payload.ParentSessionID,
				"source":            subAgentToolName,
				"task_id":           task.ID,
				"agent_id":          resolveSubAgentID(st, payload.AgentID),
			},
		})
		if err != nil {
			return "", fmt.Errorf("subagent failed: %w", err)
		}

		content = strings.TrimSpace(content)
		if content == "" {
			content = "subagent 未返回内容"
		}

		finalResult := fmt.Sprintf(
			"Subagent completed in %d iteration(s).\nSession: %s\n\n%s",
			iteration,
			payload.ChildSessionID,
			content,
		)

		if shouldDeliverSubAgentResult(payload) && payload.ParentChannel != "" && payload.ParentSessionID != "" {
			sessionKey := consts.GetSessionKey(payload.ParentChannel, payload.ParentSessionID)
			_ = st.Message().Save(&storage.Message{
				SessionID: sessionKey,
				Role:      consts.RoleAssistant,
				Content:   finalResult,
				Metadata:  fmt.Sprintf(`{"type":"subagent_result","task_id":"%s"}`, task.ID),
			})
		}

		if shouldDeliverSubAgentResult(payload) && messageBus != nil && payload.ParentChannel != "" && payload.ParentSessionID != "" {
			_ = messageBus.PublishOutbound(context.Background(), bus.OutboundMessage{
				Channel:   payload.ParentChannel,
				SessionID: payload.ParentSessionID,
				Text:      finalResult,
				Metadata: map[string]any{
					"task_id": task.ID,
					"type":    "subagent_result",
				},
			})
		}

		return finalResult, nil
	}
}

func shouldDeliverSubAgentResult(payload subAgentTaskPayload) bool {
	return normalizeSubAgentMode(payload.Mode) != "sync"
}

func normalizeSubAgentMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "sync":
		return "sync"
	default:
		return "async"
	}
}

func intArg(args map[string]any, key string, fallback int) int {
	value, ok := args[key]
	if !ok || value == nil {
		return fallback
	}
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return fallback
	}
}

func buildSubAgentSessionID(parentSessionID string) string {
	childID := uuid.NewString()
	if parentSessionID == "" {
		return "subagent:" + childID
	}
	return parentSessionID + "/subagent/" + childID
}

func buildTaskChildSessionID(payloadJSON []byte) string {
	var payload subAgentTaskPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return ""
	}
	return payload.ChildSessionID
}

func resolveSubAgentID(st *storage.Storage, requestedID string) string {
	trimmed := strings.TrimSpace(requestedID)
	if st == nil || st.Agent() == nil {
		return trimmed
	}
	if trimmed != "" {
		if agentInfo, err := st.Agent().GetByID(trimmed); err == nil && agentInfo != nil && agentInfo.Type == storage.AgentTypeSubAgent {
			return agentInfo.ID
		}
	}
	if agents, err := st.Agent().ListEnabled(); err == nil {
		for _, agentInfo := range agents {
			if agentInfo != nil && agentInfo.Type == storage.AgentTypeSubAgent {
				return agentInfo.ID
			}
		}
	}
	defaultAgent, err := st.ResolveDefaultAgent()
	if err != nil || defaultAgent == nil {
		return ""
	}
	return defaultAgent.ID
}

func stringArg(args map[string]any, key string) string {
	value, ok := args[key]
	if !ok || value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}
