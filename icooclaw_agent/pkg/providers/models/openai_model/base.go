package openai_model

// Tool 工具或工具调用。
type Tool struct {
	ID       string   `json:"id,omitempty"`    // 工具ID
	Index    int      `json:"index,omitempty"` // 工具索引
	Type     string   `json:"type,omitempty"`  // 工具类型
	Function Function `json:"function"`        // 函数定义
}

// Parameter 函数参数。
type Parameter struct {
	Type        string   `json:"type,omitempty"`        // 参数类型
	Enum        []string `json:"enum,omitempty"`        // 枚举值
	Description string   `json:"description,omitempty"` // 参数描述
}

// Parameters 函数参数。
type Parameters struct {
	Type       string               `json:"type,omitempty"`       // 参数类型
	Properties map[string]Parameter `json:"properties,omitempty"` // 参数属性
	Required   []string             `json:"required,omitempty"`   // 必填参数
	Raw        map[string]any       `json:"-"`                    // 原始 schema，供适配层兜底
}

// Function 函数定义。
type Function struct {
	Name        string         `json:"name"`                  // 函数名称
	Description string         `json:"description,omitempty"` // 函数描述
	Parameters  map[string]any `json:"parameters,omitempty"`  // 函数参数 schema
	Arguments   string         `json:"arguments,omitempty"`   // 函数参数，在调用时传递
}

// Usage token 使用。
type Usage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`     // 输入 token
	CompletionTokens int `json:"completion_tokens,omitempty"` // 输出 token
	TotalTokens      int `json:"total_tokens,omitempty"`      // 总 token
}
