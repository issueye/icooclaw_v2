package openai_responses_model

import "encoding/json"

// Usage token 使用统计。
type Usage struct {
	InputTokens           int `json:"input_tokens,omitempty"`
	OutputTokens          int `json:"output_tokens,omitempty"`
	TotalTokens           int `json:"total_tokens,omitempty"`
	InputCachedTokens     int `json:"input_cached_tokens,omitempty"`
	OutputReasoningTokens int `json:"output_reasoning_tokens,omitempty"`
}

// Error 响应错误。
type Error struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// IncompleteDetails 不完整原因。
type IncompleteDetails struct {
	Reason string `json:"reason,omitempty"`
}

// Metadata 元数据。
type Metadata map[string]string

// TextFormat 文本输出格式。
type TextFormat struct {
	Type        string          `json:"type,omitempty"`
	Name        string          `json:"name,omitempty"`
	Schema      json.RawMessage `json:"schema,omitempty"`
	Description string          `json:"description,omitempty"`
	Strict      bool            `json:"strict,omitempty"`
}

// TextConfig 文本输出配置。
type TextConfig struct {
	Format *TextFormat `json:"format,omitempty"`
}

// ReasoningConfig 推理配置。
type ReasoningConfig struct {
	Effort          string `json:"effort,omitempty"`
	Summary         string `json:"summary,omitempty"`
	GenerateSummary bool   `json:"generate_summary,omitempty"`
}

// FunctionTool 函数工具定义。
type FunctionTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	Strict      bool           `json:"strict,omitempty"`
}

// Tool 响应接口中的工具定义。
type Tool struct {
	Type     string        `json:"type"`
	Name     string        `json:"name,omitempty"`
	Function *FunctionTool `json:"function,omitempty"`
}

// ToolChoice 工具选择。
type ToolChoice struct {
	Type     string `json:"type,omitempty"`
	Name     string `json:"name,omitempty"`
	ToolName string `json:"tool_name,omitempty"`
}
