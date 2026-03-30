package platforms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers/adapter"
	"icooclaw/pkg/providers/models/openai_model"
	"icooclaw/pkg/storage"
)

type openAICompatibleProfile struct {
	defaultAPIBase string
	defaultModel   string
	chatPath       string
	headers        func(*openAICompatibleProvider) map[string]string
}

type openAICompatibleProvider struct {
	*BaseProvider
	profile openAICompatibleProfile
}

func newOpenAICompatibleProvider(
	providerName consts.ProviderType,
	cfg *storage.Provider,
	profile openAICompatibleProfile,
) *openAICompatibleProvider {
	apiBase := cfg.APIBase
	if apiBase == "" {
		apiBase = profile.defaultAPIBase
	}
	model := cfg.DefaultModel
	if model == "" {
		model = profile.defaultModel
	}
	if profile.chatPath == "" {
		profile.chatPath = "/chat/completions"
	}

	return &openAICompatibleProvider{
		BaseProvider: NewBaseProvider(providerName.ToString(), cfg.APIKey, apiBase, model),
		profile:      profile,
	}
}

func (p *openAICompatibleProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	request := adapter.ToOpenAIRequest(adapter.ToAdapterRequest(req))
	request.Stream = false
	resp, err := p.doChatRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleError(resp)
	}

	adapted, err := decodeOpenAICompatibleChatResponse(resp)
	if err != nil {
		return nil, err
	}
	return adapter.FromAdapterResponse(adapted), nil
}

func (p *openAICompatibleProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	request := adapter.ToOpenAIRequest(adapter.ToAdapterRequest(req))
	request.Stream = true
	resp, err := p.doChatRequest(ctx, request)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return p.handleError(resp)
	}

	return p.streamResponse(resp, func(content string, reasoning string, toolCalls []ToolCall, done bool) error {
		event := adapter.StreamEvent{
			Content:   content,
			Reasoning: reasoning,
			ToolCalls: adapter.ToAdapterToolCalls(toolCalls),
			Done:      done,
		}
		chunk, chunkReasoning, chunkToolCalls, chunkDone := adapter.FromAdapterStreamEvent(event)
		return callback(chunk, chunkReasoning, chunkToolCalls, chunkDone)
	})
}

func (p *openAICompatibleProvider) doChatRequest(ctx context.Context, req openai_model.Request) (*http.Response, error) {
	headers := p.profile.headers
	if headers == nil {
		return p.doRequest(ctx, http.MethodPost, p.profile.chatPath, req)
	}

	customHeaders := headers(p)
	if len(customHeaders) == 0 {
		return p.doRequest(ctx, http.MethodPost, p.profile.chatPath, req)
	}

	return p.doRequestWithHeaders(ctx, http.MethodPost, p.profile.chatPath, req, customHeaders)
}

func decodeOpenAICompatibleChatResponse(resp *http.Response) (adapter.Response, error) {
	var result openai_model.Response

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return adapter.Response{}, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return adapter.Response{}, fmt.Errorf("no choices in response")
	}

	return adapter.FromOpenAIResponse(result), nil
}
