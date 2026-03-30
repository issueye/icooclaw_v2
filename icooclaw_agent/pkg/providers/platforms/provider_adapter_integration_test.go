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

func TestGeminiProvider_Chat_UsesAdapterShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"parts": []map[string]any{
							{"text": "Gemini答案"},
							{"functionCall": map[string]any{
								"name": "weather",
								"args": map[string]any{"city": "成都"},
							}},
						},
					},
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     3,
				"candidatesTokenCount": 4,
				"totalTokenCount":      7,
			},
		})
	}))
	defer server.Close()

	provider := NewGeminiProvider(&storage.Provider{
		Type:         consts.ProviderGemini,
		APIBase:      server.URL,
		DefaultModel: "gemini-2.0-flash",
		APIKey:       "test",
	})

	resp, err := provider.Chat(context.Background(), ChatRequest{
		Model: "gemini-2.0-flash",
		Messages: []ChatMessage{
			{Role: "user", Content: "查天气"},
		},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.Content != "Gemini答案" {
		t.Fatalf("content = %q, want %q", resp.Content, "Gemini答案")
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Function.Name != "weather" {
		t.Fatalf("tool calls = %+v", resp.ToolCalls)
	}
}

func TestOllamaProvider_ChatStream_UsesAdapterShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"model\":\"llama3.2\",\"message\":{\"role\":\"assistant\",\"content\":\"第1段\"},\"done\":false}\n"))
		_, _ = w.Write([]byte("{\"model\":\"llama3.2\",\"message\":{\"role\":\"assistant\",\"content\":\"\"},\"done\":true}\n"))
	}))
	defer server.Close()

	provider := NewOllamaProvider(&storage.Provider{
		Type:         consts.ProviderOllama,
		APIBase:      server.URL,
		DefaultModel: "llama3.2",
	})

	var chunks []string
	err := provider.ChatStream(context.Background(), ChatRequest{
		Model: "llama3.2",
		Messages: []ChatMessage{
			{Role: "user", Content: "你好"},
		},
	}, func(chunk string, reasoning string, toolCalls []ToolCall, done bool) error {
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}

	if len(chunks) != 1 || chunks[0] != "第1段" {
		t.Fatalf("chunks = %+v, want [第1段]", chunks)
	}
}
