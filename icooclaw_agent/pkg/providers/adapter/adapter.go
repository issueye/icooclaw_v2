package adapter

import "context"

// StreamCallback is called for each chunk in a streaming response.
type StreamCallback func(chunk string, reasoning string, toolCalls []ToolCall, done bool) error

// Provider is the interface for LLM providers.
type Provider interface {
	// Chat sends a normalized chat request and returns a normalized response.
	Chat(ctx context.Context, req Request) (*Response, error)
	// ChatStream sends a normalized streaming request.
	ChatStream(ctx context.Context, req Request, callback StreamCallback) error
	// GetName returns the provider name.
	GetName() string
	// GetModel returns the active model.
	GetModel() string
	// SetModel sets the active model.
	SetModel(model string)
}
