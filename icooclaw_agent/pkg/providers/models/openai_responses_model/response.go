package openai_responses_model

import "encoding/json"

// OutputText 内容文本。
type OutputText struct {
	Type        string `json:"type,omitempty"`
	Text        string `json:"text,omitempty"`
	Annotations []any  `json:"annotations,omitempty"`
}

// OutputRefusal 拒答内容。
type OutputRefusal struct {
	Type    string `json:"type,omitempty"`
	Refusal string `json:"refusal,omitempty"`
}

// OutputMessage 输出消息。
type OutputMessage struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type,omitempty"`
	Status  string `json:"status,omitempty"`
	Role    string `json:"role,omitempty"`
	Content []struct {
		Type        string `json:"type,omitempty"`
		Text        string `json:"text,omitempty"`
		Refusal     string `json:"refusal,omitempty"`
		Annotations []any  `json:"annotations,omitempty"`
	} `json:"content,omitempty"`
}

// OutputFunctionCall 输出函数调用。
type OutputFunctionCall struct {
	ID        string `json:"id,omitempty"`
	Type      string `json:"type,omitempty"`
	Status    string `json:"status,omitempty"`
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// OutputReasoning 输出推理项。
type OutputReasoning struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type,omitempty"`
	Status  string `json:"status,omitempty"`
	Summary []struct {
		Type string `json:"type,omitempty"`
		Text string `json:"text,omitempty"`
	} `json:"summary,omitempty"`
}

// OutputItem 通用输出项。
type OutputItem struct {
	ID        string          `json:"id,omitempty"`
	Type      string          `json:"type,omitempty"`
	Status    string          `json:"status,omitempty"`
	Role      string          `json:"role,omitempty"`
	CallID    string          `json:"call_id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Arguments string          `json:"arguments,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	Summary   json.RawMessage `json:"summary,omitempty"`
	Raw       json.RawMessage `json:"-"`
}

// Response Responses API 响应对象。
type Response struct {
	ID                 string             `json:"id,omitempty"`
	Object             string             `json:"object,omitempty"`
	CreatedAt          int64              `json:"created_at,omitempty"`
	Status             string             `json:"status,omitempty"`
	Model              string             `json:"model,omitempty"`
	Output             []OutputItem       `json:"output,omitempty"`
	OutputText         string             `json:"output_text,omitempty"`
	Error              *Error             `json:"error,omitempty"`
	IncompleteDetails  *IncompleteDetails `json:"incomplete_details,omitempty"`
	Instructions       any                `json:"instructions,omitempty"`
	Metadata           Metadata           `json:"metadata,omitempty"`
	PreviousResponseID string             `json:"previous_response_id,omitempty"`
	Text               *TextConfig        `json:"text,omitempty"`
	Usage              Usage              `json:"usage,omitempty"`
	User               string             `json:"user,omitempty"`
}
