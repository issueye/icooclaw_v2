package anthropic_model

// Delta 流式增量块。
type Delta struct {
	Type        string `json:"type,omitempty"`
	Text        string `json:"text,omitempty"`
	Thinking    string `json:"thinking,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
	Signature   string `json:"signature,omitempty"`
}

// MessageStop 停止事件中的 message 信息。
type MessageStop struct {
	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence string `json:"stop_sequence,omitempty"`
}

// Event Anthropic SSE 事件。
type Event struct {
	Type         string       `json:"type,omitempty"`
	Index        int          `json:"index,omitempty"`
	Delta        Delta        `json:"delta,omitempty"`
	ContentBlock ContentBlock `json:"content_block,omitempty"`
	Message      MessageStop  `json:"message,omitempty"`
}
