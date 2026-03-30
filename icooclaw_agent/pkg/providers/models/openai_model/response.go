package openai_model

// BaseResp 基础响应。
type BaseResp struct {
	StatusCode int    `json:"status_code,omitempty"` // 状态码
	StatusMsg  string `json:"status_msg,omitempty"`  // 状态消息
}

// ReasoningDetail 推理详情。
type ReasoningDetail struct {
	Index  int    `json:"index,omitempty"`  // 索引
	ID     string `json:"id,omitempty"`     // ID
	Type   string `json:"type,omitempty"`   // 类型
	Format string `json:"format,omitempty"` // 格式
	Text   string `json:"text,omitempty"`   // 文本内容
}

// Message 消息。
type Message struct {
	Role             string            `json:"role,omitempty"`              // 角色
	Name             string            `json:"name,omitempty"`              // 名称
	Content          string            `json:"content,omitempty"`           // 内容
	AudioContent     string            `json:"audio_content,omitempty"`     // 音频内容
	ToolCalls        []Tool            `json:"tool_calls,omitempty"`        // 工具调用
	Reasoning        string            `json:"reasoning_content,omitempty"` // 推理内容
	ReasoningDetails []ReasoningDetail `json:"reasoning_details,omitempty"` // 推理详情
}

// Delta 流式增量消息。
type Delta struct {
	Content   string `json:"content,omitempty"`           // 内容
	Reasoning string `json:"reasoning_content,omitempty"` // 推理内容
	ToolCalls []Tool `json:"tool_calls,omitempty"`        // 工具调用
}

// Choice 选择。
type Choice struct {
	Index               int      `json:"index,omitempty"`                 // 索引
	FinishReason        string   `json:"finish_reason,omitempty"`         // 结束原因
	Message             Message  `json:"message,omitempty"`               // 完整消息
	Delta               Delta    `json:"delta,omitempty"`                 // 流式增量
	Created             int64    `json:"created,omitempty"`               // 创建时间戳
	Model               string   `json:"model,omitempty"`                 // 模型
	Object              string   `json:"object,omitempty"`                // 对象
	InputSensitive      bool     `json:"input_sensitive,omitempty"`       // 是否敏感输入
	OutputSensitive     bool     `json:"output_sensitive,omitempty"`      // 是否敏感输出
	InputSensitiveType  int      `json:"input_sensitive_type,omitempty"`  // 输入敏感类型
	OutputSensitiveType int      `json:"output_sensitive_type,omitempty"` // 输出敏感类型
	BaseResp            BaseResp `json:"base_resp,omitempty"`             // 基础响应
}

// Response 响应。
type Response struct {
	ID      string   `json:"id,omitempty"`      // 响应ID
	Model   string   `json:"model,omitempty"`   // 模型
	Object  string   `json:"object,omitempty"`  // 对象
	Created int64    `json:"created,omitempty"` // 创建时间
	Choices []Choice `json:"choices"`           // 选择
	Usage   Usage    `json:"usage,omitempty"`   // token 使用
}
