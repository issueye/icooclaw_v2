package adapter

import "encoding/json"

// Message 中间层消息模型。
type Message struct {
	Role       string
	Name       string
	Content    string
	ToolCallID string
	ToolCalls  []ToolCall
	Source     string
	Raw        json.RawMessage
}

// Tool 中间层工具定义。
type Tool struct {
	Type        string
	Name        string
	Description string
	Parameters  map[string]any
	Strict      bool
}

// ToolCall 中间层工具调用。
type ToolCall struct {
	ID        string
	Type      string
	Name      string
	Arguments string
	Source    string
	Raw       json.RawMessage
}

// Request 中间层请求模型。
type Request struct {
	Model       string
	Messages    []Message
	Tools       []Tool
	Temperature float64
	MaxTokens   int
	Stream      bool
	System      string
	Source      string
	Raw         json.RawMessage
}

// Usage 中间层 token 使用。
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Response 中间层响应模型。
type Response struct {
	ID        string
	Model     string
	Object    string
	Created   int64
	Content   string
	Reasoning string
	ToolCalls []ToolCall
	Usage     Usage
	Source    string
	Raw       json.RawMessage
}

// StreamEvent 中间层流式事件。
type StreamEvent struct {
	Type      string
	Content   string
	Reasoning string
	ToolCalls []ToolCall
	Done      bool
	Source    string
	Raw       json.RawMessage
}
