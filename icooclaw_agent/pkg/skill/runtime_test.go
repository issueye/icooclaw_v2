package skill

import (
	"context"
	"strings"
	"testing"

	"icooclaw/pkg/providers"
	"icooclaw/pkg/tools"
)

type stubLoader struct {
	info *Info
	err  error
}

func (s *stubLoader) LoadMetadata(ctx context.Context, name string) (*Metadata, error) {
	if s.info == nil {
		return nil, s.err
	}
	return &s.info.Metadata, s.err
}

func (s *stubLoader) LoadInfo(ctx context.Context, name string) (*Info, error) {
	return s.info, s.err
}

func (s *stubLoader) List(ctx context.Context) ([]*Info, error) {
	if s.info == nil {
		return nil, s.err
	}
	return []*Info{s.info}, s.err
}

type stubProvider struct {
	lastRequest providers.ChatRequest
	response    *providers.ChatResponse
	err         error
	chatFn      func(ctx context.Context, req providers.ChatRequest) (*providers.ChatResponse, error)
}

func (s *stubProvider) Chat(ctx context.Context, req providers.ChatRequest) (*providers.ChatResponse, error) {
	s.lastRequest = req
	if s.chatFn != nil {
		return s.chatFn(ctx, req)
	}
	if s.response != nil || s.err != nil {
		return s.response, s.err
	}
	return &providers.ChatResponse{Content: "ok"}, nil
}

func (s *stubProvider) ChatStream(ctx context.Context, req providers.ChatRequest, callback providers.StreamCallback) error {
	return nil
}

func (s *stubProvider) GetName() string {
	return "stub"
}

func (s *stubProvider) GetModel() string {
	return "test-model"
}

func (s *stubProvider) SetModel(model string) {}

func TestRuntimeExecute_UsesLoadedSkillInfo(t *testing.T) {
	loader := &stubLoader{
		info: &Info{
			Metadata: Metadata{
				Name:        "amap-weather",
				Title:       "AMap Weather",
				Description: "Weather lookup skill",
				Version:     "1.0.0",
			},
			Content: "# Instructions\nUse the provided weather API.",
		},
	}
	provider := &stubProvider{}
	runtime := NewRuntime(RuntimeConfig{
		Loader:    loader,
		Provider:  provider,
		ModelName: "test-model",
	})

	result := runtime.Execute(context.Background(), ExecutionRequest{
		SkillName: "amap-weather",
		Task:      "check Beijing weather",
		Context: map[string]any{
			"skill_notes": "Need concise output.",
			"user_input":  "city=Beijing",
		},
	})

	if result.Error != nil {
		t.Fatalf("Execute() error = %v", result.Error)
	}
	if !result.Success {
		t.Fatal("Execute() should succeed")
	}
	if len(provider.lastRequest.Messages) != 2 {
		t.Fatalf("Chat() messages len = %d, want 2", len(provider.lastRequest.Messages))
	}
	if provider.lastRequest.Messages[0].Content == "" {
		t.Fatal("system prompt should not be empty")
	}
	if provider.lastRequest.Messages[1].Content == "" {
		t.Fatal("user prompt should not be empty")
	}
}

type stubTool struct{}

func (s *stubTool) Name() string        { return "shell_command" }
func (s *stubTool) Description() string { return "stub shell" }
func (s *stubTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{"type": "string"},
		},
	}
}
func (s *stubTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	return &tools.Result{Success: true, Content: `{"status":"success","data":{"city":"北京","weather":"晴","temperature":"26"}}`}
}

func TestRuntimeExecute_ExecutesToolCalls(t *testing.T) {
	loader := &stubLoader{
		info: &Info{
			Metadata: Metadata{
				Name:        "amap-weather",
				Title:       "AMap Weather",
				Description: "Weather lookup skill",
				Version:     "1.0.0",
			},
			Content: "# Instructions\nUse shell_command.",
		},
	}
	provider := &stubProvider{}
	registry := tools.NewRegistry()
	registry.Register(&stubTool{})
	runtime := NewRuntime(RuntimeConfig{
		Loader:    loader,
		Provider:  provider,
		ModelName: "test-model",
		Tools:     registry,
	})

	callCount := 0
	provider.chatFn = func(ctx context.Context, req providers.ChatRequest) (*providers.ChatResponse, error) {
		callCount++
		if callCount == 1 {
			return &providers.ChatResponse{
				ToolCalls: []providers.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						}{
							Name:      "shell_command",
							Arguments: `{"command":"python3 scripts/weather.py '{\"city\":\"北京\"}'"}`,
						},
					},
				},
			}, nil
		}
		return &providers.ChatResponse{
			Content: "北京当前晴，26C。",
		}, nil
	}

	result := runtime.Execute(context.Background(), ExecutionRequest{
		SkillName: "amap-weather",
		Task:      "check Beijing weather",
	})
	if result.Error != nil {
		t.Fatalf("Execute() error = %v", result.Error)
	}
	if result.Output != "北京当前晴，26C。" {
		t.Fatalf("Output = %q, want final weather answer", result.Output)
	}
}

func TestNormalizeToolResult_UnwrapsShellCommandOutput(t *testing.T) {
	content := `{"command":"python weather.py","output":"{\"status\":\"success\",\"data\":{\"city\":\"北京\",\"weather\":\"晴\"}}","success":true}`
	result := normalizeToolResult("shell_command", content)
	if !strings.Contains(result, `"weather":"晴"`) {
		t.Fatalf("normalizeToolResult() = %q, want unwrapped command output", result)
	}
}

func TestRuntimeExecute_FallsBackToLastToolResultOnRepeatedToolCalls(t *testing.T) {
	loader := &stubLoader{
		info: &Info{
			Metadata: Metadata{
				Name:        "weather",
				Title:       "Weather",
				Description: "Weather lookup",
				Version:     "1.0.0",
			},
			Content: "# Instructions\nUse shell_command.",
		},
	}
	registry := tools.NewRegistry()
	registry.Register(&stubTool{})
	callCount := 0
	provider := &stubProvider{
		chatFn: func(ctx context.Context, req providers.ChatRequest) (*providers.ChatResponse, error) {
			callCount++
			return &providers.ChatResponse{
				ToolCalls: []providers.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						}{
							Name:      "shell_command",
							Arguments: `{"command":"weather"}`,
						},
					},
				},
			}, nil
		},
	}

	runtime := NewRuntime(RuntimeConfig{
		Loader:    loader,
		Provider:  provider,
		ModelName: "test-model",
		Tools:     registry,
	})

	result := runtime.Execute(context.Background(), ExecutionRequest{
		SkillName: "weather",
		Task:      "check weather",
	})
	if result.Error != nil {
		t.Fatalf("Execute() error = %v", result.Error)
	}
	if !result.Success {
		t.Fatal("Execute() should succeed")
	}
	if !strings.Contains(result.Output, `"weather":"晴"`) {
		t.Fatalf("Output = %q, want last tool result fallback", result.Output)
	}
	if callCount != 2 {
		t.Fatalf("callCount = %d, want 2", callCount)
	}
}
