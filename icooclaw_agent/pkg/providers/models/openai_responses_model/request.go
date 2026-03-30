package openai_responses_model

import "encoding/json"

// InputText 文本输入片段。
type InputText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// InputImage 图片输入片段。
type InputImage struct {
	Type     string `json:"type"`
	ImageURL string `json:"image_url,omitempty"`
	FileID   string `json:"file_id,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// InputFile 文件输入片段。
type InputFile struct {
	Type    string `json:"type"`
	FileID  string `json:"file_id,omitempty"`
	FileURL string `json:"file_url,omitempty"`
	Name    string `json:"filename,omitempty"`
}

// ContentItem 表示输入消息中的一个内容片段。
type ContentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	FileID   string `json:"file_id,omitempty"`
	FileURL  string `json:"file_url,omitempty"`
	Filename string `json:"filename,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// InputMessage 输入消息。
type InputMessage struct {
	Type    string        `json:"type,omitempty"`
	Role    string        `json:"role"`
	Content []ContentItem `json:"content"`
}

// InputFunctionCallOutput 函数调用结果输入。
type InputFunctionCallOutput struct {
	Type   string          `json:"type"`
	CallID string          `json:"call_id"`
	Output json.RawMessage `json:"output"`
}

// InputItem 表示 Responses API 的一个输入项。
type InputItem struct {
	Type    string          `json:"type,omitempty"`
	Role    string          `json:"role,omitempty"`
	Content []ContentItem   `json:"content,omitempty"`
	CallID  string          `json:"call_id,omitempty"`
	Output  json.RawMessage `json:"output,omitempty"`
}

// Request Responses API 请求体。
type Request struct {
	Model              string           `json:"model"`
	Input              any              `json:"input,omitempty"`
	Instructions       any              `json:"instructions,omitempty"`
	PreviousResponseID string           `json:"previous_response_id,omitempty"`
	MaxOutputTokens    int              `json:"max_output_tokens,omitempty"`
	Temperature        float64          `json:"temperature,omitempty"`
	TopP               float64          `json:"top_p,omitempty"`
	Stream             bool             `json:"stream,omitempty"`
	Background         bool             `json:"background,omitempty"`
	Store              *bool            `json:"store,omitempty"`
	Metadata           Metadata         `json:"metadata,omitempty"`
	Text               *TextConfig      `json:"text,omitempty"`
	Reasoning          *ReasoningConfig `json:"reasoning,omitempty"`
	Tools              []Tool           `json:"tools,omitempty"`
	ToolChoice         any              `json:"tool_choice,omitempty"`
	ParallelToolCalls  *bool            `json:"parallel_tool_calls,omitempty"`
	User               string           `json:"user,omitempty"`
}
