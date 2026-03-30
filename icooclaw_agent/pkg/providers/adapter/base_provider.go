package adapter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"icooclaw/pkg/errors"
	"icooclaw/pkg/providers/models/openai_model"
	"icooclaw/pkg/providers/protocol"
)

// BaseProvider provides common functionality for providers.
type BaseProvider struct {
	name       string
	apiKey     string
	apiBase    string
	model      string
	httpClient *http.Client
}

// NewBaseProvider creates a new BaseProvider.
func NewBaseProvider(name, apiKey, apiBase, model string) *BaseProvider {
	return &BaseProvider{
		name:    name,
		apiKey:  apiKey,
		apiBase: apiBase,
		model:   model,
		httpClient: &http.Client{
			Timeout: 300 * time.Second,
		},
	}
}

func (p *BaseProvider) GetName() string {
	return p.name
}

func (p *BaseProvider) GetModel() string {
	return p.model
}

func (p *BaseProvider) SetModel(model string) {
	p.model = model
}

func (p *BaseProvider) DoRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := p.apiBase + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

func (p *BaseProvider) DoRequestWithHeaders(ctx context.Context, method, path string, body any, headers map[string]string) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := p.apiBase + path
	// 打印请求头
	slog.Debug("请求头：", slog.Any("headers", headers))
	// 打印请求地址
	slog.Info("请求地址：", slog.String("path", url))
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

func (p *BaseProvider) HandleError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case 401:
		return errors.NewFailoverError(errors.FailoverAuth, p.name, "", resp.StatusCode, fmt.Errorf("auth failed: %s", string(body)))
	case 429:
		return errors.NewFailoverError(errors.FailoverRateLimit, p.name, "", resp.StatusCode, fmt.Errorf("rate limited: %s", string(body)))
	case 500, 502, 503, 504:
		return errors.NewFailoverError(errors.FailoverTimeout, p.name, "", resp.StatusCode, fmt.Errorf("server error: %s", string(body)))
	default:
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}
}

func (p *BaseProvider) StreamResponse(resp *http.Response, callback protocol.StreamCallback) error {
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		content, reasoning, toolCalls, done, err := parseStreamChunk(data)
		if err != nil {
			continue
		}

		if err := callback(content, reasoning, toolCalls, done); err != nil {
			return err
		}

		if done {
			return nil
		}
	}

	return scanner.Err()
}

func (p *BaseProvider) APIKey() string {
	return p.apiKey
}

func (p *BaseProvider) APIBase() string {
	return p.apiBase
}

func (p *BaseProvider) HTTPClient() *http.Client {
	return p.httpClient
}

func parseStreamChunk(data string) (content string, reasoning string, toolCalls []protocol.ToolCall, done bool, err error) {
	if data == "[DONE]" {
		return "", "", nil, true, nil
	}

	var chunk openai_model.Response
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return "", "", nil, false, err
	}

	event := FromOpenAIStreamResponse(chunk)
	content, reasoning, toolCalls, done = FromAdapterStreamEvent(event)
	if len(toolCalls) > 0 {
		for i := range toolCalls {
			if toolCalls[i].ID == "" {
				toolCalls[i].ID = fmt.Sprintf("stream_index:%d", i)
			}
		}
	}

	return content, reasoning, toolCalls, done, nil
}
