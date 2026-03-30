package platforms

import (
	"context"
	"net/http"

	"icooclaw/pkg/providers/adapter"
	"icooclaw/pkg/providers/protocol"
)

type Provider = protocol.Provider
type ChatMessage = protocol.ChatMessage
type ChatRequest = protocol.ChatRequest
type Tool = protocol.Tool
type ToolCall = protocol.ToolCall
type ChatResponse = protocol.ChatResponse
type Usage = protocol.Usage
type StreamCallback = protocol.StreamCallback

type BaseProvider struct {
	*adapter.BaseProvider
}

func NewBaseProvider(name, apiKey, apiBase, model string) *BaseProvider {
	return &BaseProvider{BaseProvider: adapter.NewBaseProvider(name, apiKey, apiBase, model)}
}

func (p *BaseProvider) doRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	return p.BaseProvider.DoRequest(ctx, method, path, body)
}

func (p *BaseProvider) doRequestWithHeaders(ctx context.Context, method, path string, body any, headers map[string]string) (*http.Response, error) {
	return p.BaseProvider.DoRequestWithHeaders(ctx, method, path, body, headers)
}

func (p *BaseProvider) handleError(resp *http.Response) error {
	return p.BaseProvider.HandleError(resp)
}

func (p *BaseProvider) streamResponse(resp *http.Response, callback StreamCallback) error {
	return p.BaseProvider.StreamResponse(resp, callback)
}

func (p *BaseProvider) APIKey() string {
	return p.BaseProvider.APIKey()
}

func (p *BaseProvider) APIBase() string {
	return p.BaseProvider.APIBase()
}

func (p *BaseProvider) HTTPClient() *http.Client {
	return p.BaseProvider.HTTPClient()
}
