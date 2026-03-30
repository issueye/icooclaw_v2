package openai_model

// RequestTool 请求工具。
type RequestTool struct {
	Type     string   `json:"type"`     // 类型
	Function Function `json:"function"` // 函数
}

// ChatMessage 消息在聊天中的角色。
type ChatMessage struct {
	Role       string `json:"role"`                   // 角色
	Name       string `json:"name,omitempty"`         // 名称
	Content    string `json:"content"`                // 内容
	ToolCallID string `json:"tool_call_id,omitempty"` // 工具调用 ID
	ToolCalls  []Tool `json:"tool_calls,omitempty"`   // 工具调用列表
}

// ExtraBody 额外体。
type ExtraBody struct {
	ReasoningSplit bool `json:"reasoning_split,omitempty"` // 是否开启推理分割
}

// Request 聊天请求。
type Request struct {
	Model       string        `json:"model"`                 // 模型名称
	Stream      bool          `json:"stream,omitempty"`      // 是否流式响应
	Tools       []RequestTool `json:"tools,omitempty"`       // 工具列表
	Messages    []ChatMessage `json:"messages"`              // 消息列表
	Temperature float64       `json:"temperature,omitempty"` // 温度
	MaxTokens   int           `json:"max_tokens,omitempty"`  // 最大 token 数
	ExtraBody   ExtraBody     `json:"extra_body,omitempty"`  // 额外体
}
