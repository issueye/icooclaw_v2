// Package providers provides LLM provider abstraction for icooclaw.
package providers

import (
	"testing"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

func TestGetModelInfo(t *testing.T) {
	tests := []struct {
		modelID   string
		wantName  string
		wantFound bool
	}{
		{"gpt-4o", "GPT-4o", true},
		{"claude-3-5-sonnet-20241022", "Claude 3.5 Sonnet", true},
		{"claude-3.5-sonnet", "Claude 3.5 Sonnet", true},
		{"deepseek-chat", "DeepSeek Chat", true},
		{"unknown-model", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			info := GetModelInfo(tt.modelID)
			if tt.wantFound {
				if info == nil {
					t.Errorf("Expected to find model %s", tt.modelID)
					return
				}
				if info.Name != tt.wantName {
					t.Errorf("Expected name %s, got %s", tt.wantName, info.Name)
				}
			} else if info != nil {
				t.Errorf("Expected not to find model %s", tt.modelID)
			}
		})
	}
}

func TestListModels(t *testing.T) {
	models := ListModels()
	if len(models) == 0 {
		t.Error("Expected at least one model")
	}
}

func TestListModelsByProvider(t *testing.T) {
	openaiModels := ListModelsByProvider("openai")
	if len(openaiModels) == 0 {
		t.Error("Expected at least one OpenAI model")
	}

	anthropicModels := ListModelsByProvider("anthropic")
	if len(anthropicModels) == 0 {
		t.Error("Expected at least one Anthropic model")
	}
}

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		modelID      string
		inputTokens  int
		outputTokens int
		wantError    bool
	}{
		{"gpt-4o", 1000, 500, false},
		{"claude-3-5-sonnet-20241022", 1000, 500, false},
		{"unknown-model", 1000, 500, true},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			cost, err := CalculateCost(tt.modelID, tt.inputTokens, tt.outputTokens)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if cost <= 0 {
				t.Errorf("Expected positive cost, got %f", cost)
			}
		})
	}
}

func TestManager_RegisterFactory(t *testing.T) {
	manager := NewManager(nil, nil)

	factories := []providerFactoryKey{
		{providerType: consts.ProviderOpenAI, protocol: consts.ProtocolOpenAI},
		{providerType: consts.ProviderAnthropic, protocol: consts.ProtocolAnthropic},
		{providerType: consts.ProviderMiniMax, protocol: consts.ProtocolOpenAI},
		{providerType: consts.ProviderMiniMax, protocol: consts.ProtocolAnthropic},
		{providerType: consts.ProviderDeepSeek, protocol: consts.ProtocolOpenAI},
		{providerType: consts.ProviderOpenRouter, protocol: consts.ProtocolOpenAI},
		{providerType: consts.ProviderGemini, protocol: consts.ProtocolOpenAI},
		{providerType: consts.ProviderMistral, protocol: consts.ProtocolOpenAI},
		{providerType: consts.ProviderGroq, protocol: consts.ProtocolOpenAI},
		{providerType: consts.ProviderOllama, protocol: consts.ProtocolOpenAI},
		{providerType: consts.ProviderMoonshot, protocol: consts.ProtocolOpenAI},
		{providerType: consts.ProviderQwen, protocol: consts.ProtocolOpenAI},
		{providerType: consts.ProviderSiliconFlow, protocol: consts.ProtocolOpenAI},
	}

	for _, key := range factories {
		if _, ok := manager.factories[key]; !ok {
			t.Errorf("Expected factory for %s/%s", key.providerType, key.protocol)
		}
	}
}

func TestManager_CreateProvider(t *testing.T) {
	manager := NewManager(nil, nil)

	tests := []struct {
		providerType string
		protocol     consts.ProviderProtocol
		wantError    bool
	}{
		{"openai", consts.ProtocolOpenAI, false},
		{"anthropic", consts.ProtocolAnthropic, false},
		{"minimax", consts.ProtocolOpenAI, false},
		{"minimax", consts.ProtocolAnthropic, false},
		{"deepseek", consts.ProtocolOpenAI, false},
		{"unknown", consts.ProtocolOpenAI, true},
		{"openai", "", true},
		{"anthropic", consts.ProtocolOpenAI, true},
	}

	for _, tt := range tests {
		t.Run(tt.providerType+"-"+tt.protocol.String(), func(t *testing.T) {
			cfg := &storage.Provider{
				Name:         "test",
				Type:         consts.ToProviderType(tt.providerType),
				Protocol:     tt.protocol,
				APIKey:       "test-key",
				DefaultModel: "test-model",
			}

			provider, err := manager.CreateProvider(cfg)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if provider == nil {
				t.Error("Expected provider")
			}
		})
	}
}

func TestManager_RegisterAndGet(t *testing.T) {
	manager := NewManager(nil, nil)

	cfg := &storage.Provider{
		Name:         "test-openai",
		Type:         "openai",
		Protocol:     consts.ProtocolOpenAI,
		APIKey:       "test-key",
		DefaultModel: "gpt-4o",
	}

	provider, err := manager.CreateProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	manager.Register("my-openai", provider)

	got, err := manager.Get("my-openai")
	if err != nil {
		t.Fatalf("Failed to get provider: %v", err)
	}

	if got.GetName() != "openai" {
		t.Errorf("Expected name 'openai', got %s", got.GetName())
	}

	names := manager.List()
	if len(names) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(names))
	}
}

func TestModelInfo_SupportsVision(t *testing.T) {
	tests := []struct {
		modelID    string
		wantVision bool
	}{
		{"gpt-4o", true},
		{"gpt-4o-mini", true},
		{"deepseek-chat", false},
		{"gemini-2.0-flash", true},
		{"claude-3-5-sonnet-20241022", true},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			info := GetModelInfo(tt.modelID)
			if info == nil {
				t.Fatalf("Model not found: %s", tt.modelID)
			}
			if info.SupportsVision != tt.wantVision {
				t.Errorf("Expected SupportsVision=%v, got %v", tt.wantVision, info.SupportsVision)
			}
		})
	}
}

func TestModelInfo_SupportsTools(t *testing.T) {
	tests := []struct {
		modelID   string
		wantTools bool
	}{
		{"gpt-4o", true},
		{"o1", false},
		{"deepseek-chat", true},
		{"claude-3-5-sonnet-20241022", true},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			info := GetModelInfo(tt.modelID)
			if info == nil {
				t.Fatalf("Model not found: %s", tt.modelID)
			}
			if info.SupportsTools != tt.wantTools {
				t.Errorf("Expected SupportsTools=%v, got %v", tt.wantTools, info.SupportsTools)
			}
		})
	}
}

func TestModelInfo_ContextWindow(t *testing.T) {
	tests := []struct {
		modelID     string
		wantContext int
	}{
		{"gpt-4o", 128000},
		{"claude-3-5-sonnet-20241022", 200000},
		{"gemini-1.5-pro", 2097152},
		{"deepseek-chat", 64000},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			info := GetModelInfo(tt.modelID)
			if info == nil {
				t.Fatalf("Model not found: %s", tt.modelID)
			}
			if info.ContextWindow != tt.wantContext {
				t.Errorf("Expected ContextWindow=%d, got %d", tt.wantContext, info.ContextWindow)
			}
		})
	}
}
