package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

func TestOpenAICompatibleProvider_Chat_ParsesReasoningAndToolCalls(t *testing.T) {
	var captured ChatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %s, want /chat/completions", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "resp-1",
			"model": "deepseek-chat",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":              "assistant",
						"content":           "最终答案",
						"reasoning_content": "推理过程",
						"tool_calls": []map[string]any{
							{
								"id":   "call_1",
								"type": "function",
								"function": map[string]any{
									"name":      "weather",
									"arguments": `{"city":"成都"}`,
								},
							},
						},
					},
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     11,
				"completion_tokens": 7,
				"total_tokens":      18,
			},
		})
	}))
	defer server.Close()

	provider, ok := NewDeepSeekProvider(&storage.Provider{
		Name:         "deepseek",
		Type:         consts.ProviderDeepSeek,
		APIKey:       "test-key",
		APIBase:      server.URL,
		DefaultModel: "deepseek-chat",
	}).(*DeepSeekProvider)
	if !ok {
		t.Fatalf("expected DeepSeekProvider, got %T", provider)
	}

	resp, err := provider.Chat(context.Background(), ChatRequest{
		Model: "deepseek-chat",
		Messages: []ChatMessage{
			{Role: "user", Content: "今天天气如何"},
		},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if captured.Stream {
		t.Fatal("expected sync request to set stream=false")
	}
	if resp.Content != "最终答案" {
		t.Fatalf("content = %q, want %q", resp.Content, "最终答案")
	}
	if resp.Reasoning != "推理过程" {
		t.Fatalf("reasoning = %q, want %q", resp.Reasoning, "推理过程")
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("tool call count = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Function.Name != "weather" {
		t.Fatalf("tool name = %q, want %q", resp.ToolCalls[0].Function.Name, "weather")
	}
	if resp.Usage.TotalTokens != 18 {
		t.Fatalf("total_tokens = %d, want %d", resp.Usage.TotalTokens, 18)
	}
}

func TestOpenAICompatibleProvider_ChatStream_ParsesChunks(t *testing.T) {
	var captured ChatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"先分析\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"答案\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	provider, ok := NewQwenProvider(&storage.Provider{
		Name:         "qwen",
		Type:         consts.ProviderQwen,
		APIKey:       "test-key",
		APIBase:      server.URL,
		DefaultModel: "qwen-plus",
	}).(*QwenProvider)
	if !ok {
		t.Fatalf("expected QwenProvider, got %T", provider)
	}

	var chunks []struct {
		content   string
		reasoning string
		done      bool
	}
	err := provider.ChatStream(context.Background(), ChatRequest{
		Model: "qwen-plus",
		Messages: []ChatMessage{
			{Role: "user", Content: "请分析并回答"},
		},
	}, func(content string, reasoning string, toolCalls []ToolCall, done bool) error {
		chunks = append(chunks, struct {
			content   string
			reasoning string
			done      bool
		}{
			content:   content,
			reasoning: reasoning,
			done:      done,
		})
		return nil
	})
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}

	if !captured.Stream {
		t.Fatal("expected stream request to set stream=true")
	}
	if len(chunks) != 3 {
		t.Fatalf("chunk count = %d, want %d", len(chunks), 3)
	}
	if chunks[0].reasoning != "先分析" {
		t.Fatalf("first reasoning = %q, want %q", chunks[0].reasoning, "先分析")
	}
	if chunks[1].content != "答案" {
		t.Fatalf("second content = %q, want %q", chunks[1].content, "答案")
	}
	if !chunks[2].done {
		t.Fatal("expected final done chunk")
	}
}

func TestNewQwenProvider_CodingPlanUsesDedicatedEndpoint(t *testing.T) {
	provider, ok := NewQwenProvider(&storage.Provider{
		Name: "qwen-coding-plan",
		Type: consts.ProviderQwenCodingPlan,
	}).(*QwenProvider)
	if !ok {
		t.Fatalf("expected QwenProvider, got %T", provider)
	}

	if provider.GetName() != consts.ProviderQwenCodingPlan.ToString() {
		t.Fatalf("provider name = %q, want %q", provider.GetName(), consts.ProviderQwenCodingPlan.ToString())
	}
	if provider.APIBase() != "https://coding.dashscope.aliyuncs.com/v1" {
		t.Fatalf("apiBase = %q, want %q", provider.APIBase(), "https://coding.dashscope.aliyuncs.com/v1")
	}
}
