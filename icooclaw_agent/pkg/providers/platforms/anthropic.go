package platforms

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers/adapter"
	"icooclaw/pkg/providers/models/anthropic_model"
	"icooclaw/pkg/storage"
)

// AnthropicProvider implements Provider for Anthropic Claude.
type AnthropicProvider struct {
	*BaseProvider
	chatPath string
}

func newAnthropicProvider(
	providerName consts.ProviderType,
	cfg *storage.Provider,
	defaultAPIBase string,
	defaultModel string,
) *AnthropicProvider {
	apiBase := cfg.APIBase
	if apiBase == "" {
		apiBase = defaultAPIBase
	}
	model := cfg.DefaultModel
	if model == "" {
		model = defaultModel
	}

	return &AnthropicProvider{
		BaseProvider: NewBaseProvider(providerName.ToString(), cfg.APIKey, apiBase, model),
		chatPath:     resolveAnthropicChatPath(cfg, apiBase),
	}
}

// NewAnthropicProvider creates a new Anthropic provider.
func NewAnthropicProvider(cfg *storage.Provider) Provider {
	return newAnthropicProvider(
		consts.ProviderAnthropic,
		cfg,
		"https://api.anthropic.com/v1",
		"claude-3-5-sonnet-20241022",
	)
}

// Chat sends a chat request to Anthropic.
func (p *AnthropicProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	adaptedReq := adapter.ToAdapterRequest(req)
	anthropicReq := adapter.ToAnthropicRequest(adaptedReq)
	if anthropicReq.MaxTokens == 0 {
		anthropicReq.MaxTokens = 4096
	}

	headers := map[string]string{
		"x-api-key":         p.APIKey(),
		"anthropic-version": "2023-06-01",
	}

	resp, err := p.doRequestWithHeaders(ctx, http.MethodPost, p.chatPath, anthropicReq, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleError(resp)
	}

	var result anthropic_model.Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return adapter.FromAdapterResponse(adapter.FromAnthropicResponse(result)), nil
}

// ChatStream sends a streaming chat request to Anthropic.
func (p *AnthropicProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	adaptedReq := adapter.ToAdapterRequest(req)
	anthropicReq := adapter.ToAnthropicRequest(adaptedReq)
	anthropicReq.Stream = true
	if anthropicReq.MaxTokens == 0 {
		anthropicReq.MaxTokens = 4096
	}

	headers := map[string]string{
		"x-api-key":         p.APIKey(),
		"anthropic-version": "2023-06-01",
	}
	resp, err := p.doRequestWithHeaders(ctx, http.MethodPost, p.chatPath, anthropicReq, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return p.handleError(resp)
	}

	streamBlocks := map[int]anthropic_model.ContentBlock{}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		var event anthropic_model.Event
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_start":
			streamBlocks[event.Index] = event.ContentBlock
		case "content_block_delta":
			if block, ok := streamBlocks[event.Index]; ok {
				event.ContentBlock = block
			}
			streamEvent := adapter.FromAnthropicEvent(event)
			content, reasoning, toolCalls, done := adapter.FromAdapterStreamEvent(streamEvent)
			if content == "" && reasoning == "" && len(toolCalls) == 0 && !done {
				continue
			}
			if err := callback(content, reasoning, toolCalls, done); err != nil {
				return err
			}
		case "message_stop":
			content, reasoning, toolCalls, done := adapter.FromAdapterStreamEvent(adapter.FromAnthropicEvent(event))
			if err := callback(content, reasoning, toolCalls, done); err != nil {
				return err
			}
			return nil
		}
	}

	return scanner.Err()
}

func resolveAnthropicChatPath(cfg *storage.Provider, apiBase string) string {
	if path := readAnthropicChatPathOverride(cfg); path != "" {
		return path
	}

	normalizedBase := strings.ToLower(strings.TrimSpace(apiBase))
	switch {
	case strings.HasSuffix(normalizedBase, "/v1/messages"), strings.HasSuffix(normalizedBase, "/messages"):
		return ""
	case strings.HasSuffix(normalizedBase, "/v1"), strings.HasSuffix(normalizedBase, "/v1/"):
		return "/messages"
	default:
		return "/v1/messages"
	}
}

func readAnthropicChatPathOverride(cfg *storage.Provider) string {
	if cfg == nil {
		return ""
	}
	if path := readStringValue(cfg.Metadata, "chat_path"); path != "" {
		return normalizeProviderPath(path)
	}
	if strings.TrimSpace(cfg.Config) == "" {
		return ""
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(cfg.Config), &raw); err != nil {
		return ""
	}
	return normalizeProviderPath(readStringValue(raw, "chat_path"))
}

func readStringValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func normalizeProviderPath(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}
