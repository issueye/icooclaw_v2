package openai_responses_model

import "encoding/json"

// StreamEvent Responses API 流式事件。
type StreamEvent struct {
	Type           string          `json:"type,omitempty"`
	SequenceNumber int64           `json:"sequence_number,omitempty"`
	ResponseID     string          `json:"response_id,omitempty"`
	OutputIndex    int             `json:"output_index,omitempty"`
	ItemID         string          `json:"item_id,omitempty"`
	Delta          string          `json:"delta,omitempty"`
	Text           string          `json:"text,omitempty"`
	Part           json.RawMessage `json:"part,omitempty"`
	Item           json.RawMessage `json:"item,omitempty"`
	Response       *Response       `json:"response,omitempty"`
	Error          *Error          `json:"error,omitempty"`
}
