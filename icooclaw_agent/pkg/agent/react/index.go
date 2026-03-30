package react

import (
	"context"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/memory"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/skill"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
	"log/slog"
)

// StreamChunk 表示流式响应的一个数据块。
type StreamChunk struct {
	Content     string `json:"content,omitempty"`      // 内容
	Reasoning   string `json:"reasoning,omitempty"`    // 推理过程
	ToolCallID  string `json:"tool_call_id,omitempty"` // 工具调用 ID
	ToolName    string `json:"tool_name,omitempty"`    // 工具名称
	ToolArgs    string `json:"tool_args,omitempty"`    // 工具参数
	ToolResult  string `json:"tool_result,omitempty"`  // 工具结果
	TotalTokens int    `json:"total_tokens,omitempty"` // token 使用量
	Iteration   int    `json:"iteration,omitempty"`    // 迭代次数
	Done        bool   `json:"done,omitempty"`         // 是否完成
	Error       error  `json:"error,omitempty"`        // 错误信息
}

// StreamCallback 流式响应的回调函数。
type StreamCallback func(chunk StreamChunk) error

type MessageTraceItem struct {
	Type       string `json:"type"`
	Content    string `json:"content,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	ToolName   string `json:"tool_name,omitempty"`
	ToolArgs   string `json:"tool_args,omitempty"`
	ToolResult string `json:"tool_result,omitempty"`
	Iteration  int    `json:"iteration,omitempty"`
}

type AssistantMessageMetadata struct {
	Type             string             `json:"type,omitempty"`
	Iteration        int                `json:"iteration,omitempty"`
	ReasoningContent string             `json:"reasoning_content,omitempty"`
	TraceItems       []MessageTraceItem `json:"trace_items,omitempty"`
}

type ReActAgent struct {
	tools           *tools.Registry    // 工具注册表
	memory          memory.Loader      // 内存加载器
	skills          skill.Loader       // 工具加载器
	storage         *storage.Storage   // 存储管理
	bus             *bus.MessageBus    // 消息总线
	providerManager *providers.Manager // 提供商管理器
	logger          *slog.Logger       // 日志记录器
	hooks           ReactHooks         // React钩子接口

	// 运行配置
	maxToolIterations int // 最大工具迭代次数
}

type Dependencies struct {
	Tools             *tools.Registry
	Memory            memory.Loader
	Skills            skill.Loader
	Storage           *storage.Storage
	Bus               *bus.MessageBus
	ProviderManager   *providers.Manager
	Logger            *slog.Logger
	MaxToolIterations int
}

func NewReActAgent(ctx context.Context, hooks ReactHooks, deps Dependencies) (*ReActAgent, error) {
	a := &ReActAgent{
		tools:             deps.Tools,
		memory:            deps.Memory,
		skills:            deps.Skills,
		storage:           deps.Storage,
		bus:               deps.Bus,
		providerManager:   deps.ProviderManager,
		logger:            deps.Logger,
		hooks:             hooks,
		maxToolIterations: deps.MaxToolIterations,
	}
	if a.logger == nil {
		a.logger = slog.Default()
	}
	if a.maxToolIterations <= 0 {
		a.maxToolIterations = consts.DEFAULT_TOOL_ITERATIONS
	}

	var err error
	if a.hooks != nil {
		a, err = a.hooks.OnCreateAgent(ctx, a)
		if err != nil {
			return nil, err
		}
	}

	return a, nil
}

func NewReActAgentNoHooks(ctx context.Context, deps Dependencies) (*ReActAgent, error) {
	return NewReActAgent(ctx, nil, deps)
}
