package tool

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/storage"
)

type mockProvider struct {
	chatCalled    bool
	chatError     error
	chatResponse  string
	chatModelName string
}

func (m *mockProvider) Chat(ctx context.Context, req providers.ChatRequest) (*providers.ChatResponse, error) {
	m.chatCalled = true
	return &providers.ChatResponse{Content: m.chatResponse}, m.chatError
}

func (m *mockProvider) ChatStream(ctx context.Context, req providers.ChatRequest, callback providers.StreamCallback) error {
	return nil
}

func (m *mockProvider) GetName() string {
	return "mock"
}

func (m *mockProvider) GetModel() string {
	return m.chatModelName
}

func (m *mockProvider) SetModel(model string) {
	m.chatModelName = model
}

func TestExecuteSkillTool_Name(t *testing.T) {
	tool := NewExecuteSkillTool("/workspace", nil, nil, nil, nil)

	if tool.Name() != "skill_execute" {
		t.Errorf("Name() = %q, want skill_execute", tool.Name())
	}
}

func TestExecuteSkillTool_Description(t *testing.T) {
	tool := NewExecuteSkillTool("/workspace", nil, nil, nil, nil)

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() should not be empty")
	}

	if desc != "Execute an installed skill to perform a specific task. The skill provides specialized instructions and workflows for the LLM to follow. Use this when a user asks for something that matches a skill's purpose." {
		t.Errorf("Description() = %q", desc)
	}
}

func TestExecuteSkillTool_Parameters(t *testing.T) {
	tool := NewExecuteSkillTool("/workspace", nil, nil, nil, nil)

	params := tool.Parameters()

	if params["type"] != "object" {
		t.Errorf("Parameters() type = %q, want object", params["type"])
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Parameters() properties is not a map")
	}

	if _, ok := props["skill_name"]; !ok {
		t.Error("Parameters() missing skill_name property")
	}

	if _, ok := props["task"]; !ok {
		t.Error("Parameters() missing task property")
	}

	if _, ok := props["context"]; !ok {
		t.Error("Parameters() missing context property")
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("Parameters() required is not a slice")
	}

	found := false
	for _, r := range required {
		if r == "skill_name" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Parameters() required should include skill_name")
	}
}

func TestListSkillsTool_Name(t *testing.T) {
	tool := NewListSkillsTool(nil, nil)

	if tool.Name() != "skill_list" {
		t.Errorf("Name() = %q, want skill_list", tool.Name())
	}
}

func TestListSkillsTool_Description(t *testing.T) {
	tool := NewListSkillsTool(nil, nil)

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() should not be empty")
	}
}

func TestListSkillsTool_Parameters(t *testing.T) {
	tool := NewListSkillsTool(nil, nil)

	params := tool.Parameters()

	if params["type"] != "object" {
		t.Errorf("Parameters() type = %q, want object", params["type"])
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Parameters() properties is not a map")
	}

	if len(props) != 0 {
		t.Errorf("Parameters() properties should be empty, got %d properties", len(props))
	}
}

func TestGetSkillInfoTool_Name(t *testing.T) {
	tool := NewGetSkillInfoTool(nil, nil)

	if tool.Name() != "skill_info" {
		t.Errorf("Name() = %q, want skill_info", tool.Name())
	}
}

func TestGetSkillInfoTool_Description(t *testing.T) {
	tool := NewGetSkillInfoTool(nil, nil)

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() should not be empty")
	}
}

func TestGetSkillInfoTool_Parameters(t *testing.T) {
	tool := NewGetSkillInfoTool(nil, nil)

	params := tool.Parameters()

	if params["type"] != "object" {
		t.Errorf("Parameters() type = %q, want object", params["type"])
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Parameters() properties is not a map")
	}

	if _, ok := props["skill_name"]; !ok {
		t.Error("Parameters() missing skill_name property")
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("Parameters() required is not a slice")
	}

	found := false
	for _, r := range required {
		if r == "skill_name" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Parameters() required should include skill_name")
	}
}

func TestExecuteSkillTool_Execute_MissingSkillName(t *testing.T) {
	tool := NewExecuteSkillTool("/workspace", nil, nil, nil, nil)

	result := tool.Execute(context.Background(), map[string]any{})

	if result.Success {
		t.Error("Execute() should return error for missing skill_name")
	}

	if result.Error == nil || result.Error.Error() == "" {
		t.Error("Execute() should return error message")
	}
}

func TestExecuteSkillTool_Execute_MissingTask(t *testing.T) {
	tool := NewExecuteSkillTool("/workspace", nil, nil, nil, nil)

	result := tool.Execute(context.Background(), map[string]any{
		"skill_name": "test-skill",
	})

	if result.Success {
		t.Error("Execute() should return error for missing task")
	}

	if result.Error == nil || result.Error.Error() == "" {
		t.Error("Execute() should return error message")
	}
}

func TestGetSkillInfoTool_Execute_MissingSkillName(t *testing.T) {
	tool := NewGetSkillInfoTool(nil, nil)

	result := tool.Execute(context.Background(), map[string]any{})

	if result.Success {
		t.Error("Execute() should return error for missing skill_name")
	}

	if result.Error == nil || result.Error.Error() == "" {
		t.Error("Execute() should return error message")
	}
}

func TestExecuteSkillTool_Execute_InvalidSkillNameType(t *testing.T) {
	tool := NewExecuteSkillTool("/workspace", nil, nil, nil, nil)

	result := tool.Execute(context.Background(), map[string]any{
		"skill_name": 123,
		"task":       "do something",
	})

	if result.Success {
		t.Error("Execute() should return error for non-string skill_name")
	}
}

func TestExecuteSkillTool_Execute_EmptySkillName(t *testing.T) {
	tool := NewExecuteSkillTool("/workspace", nil, nil, nil, nil)

	result := tool.Execute(context.Background(), map[string]any{
		"skill_name": "",
		"task":       "do something",
	})

	if result.Success {
		t.Error("Execute() should return error for empty skill_name")
	}
}

func TestListSkillsTool_Execute_EmptyContext(t *testing.T) {
	tool := NewListSkillsTool(nil, nil)

	result := tool.Execute(context.Background(), map[string]any{})

	if !result.Success {
		t.Errorf("Execute() should succeed with no skills: %v", result.Error)
	}

	if result.Content == "" {
		t.Error("Execute() should return message about no skills")
	}
}

func TestFormatSkillExecutionOutput(t *testing.T) {
	result := formatSkillExecutionOutput("Weather [weather]", "北京当前晴，26C。")
	if !strings.Contains(result, "技能: Weather [weather]") {
		t.Fatalf("formatted output = %q, want skill header", result)
	}
	if !strings.Contains(result, "北京当前晴，26C。") {
		t.Fatalf("formatted output = %q, want body", result)
	}
}

func TestExecuteSkillTool_ResolveProvider_UsesSkillDefaultModel(t *testing.T) {
	store := newTestSkillToolStorage(t)
	saveTestProvider(t, store, &storage.Provider{
		Name:         "qianfan",
		Type:         consts.ProviderAnthropic,
		Protocol:     consts.ProtocolAnthropic,
		APIBase:      "https://qianfan.baidubce.com/anthropic/coding",
		APIKey:       "test-key",
		DefaultModel: "glm-5",
		Enabled:      true,
	})
	saveTestProvider(t, store, &storage.Provider{
		Name:         "openai-main",
		Type:         consts.ProviderOpenAI,
		Protocol:     consts.ProtocolOpenAI,
		APIBase:      "https://api.openai.com/v1",
		APIKey:       "test-key",
		DefaultModel: "gpt-4o-mini",
		Enabled:      true,
	})
	saveTestParam(t, store, consts.DEFAULT_MODEL_KEY, "qianfan/glm-5")
	saveTestParam(t, store, consts.SKILL_DEFAULT_MODEL_KEY, "openai-main/gpt-4o-mini")

	manager := providers.NewManager(store, nil)
	tool := NewExecuteSkillTool("/workspace", store, manager, nil, nil)

	provider, modelName, err := tool.resolveProvider()
	if err != nil {
		t.Fatalf("resolveProvider() error = %v", err)
	}
	if provider.GetName() != "openai" {
		t.Fatalf("provider name = %s, want openai", provider.GetName())
	}
	if modelName != "gpt-4o-mini" {
		t.Fatalf("model name = %s, want gpt-4o-mini", modelName)
	}
}

func TestExecuteSkillTool_ResolveProvider_UsesSupportedCodingEndpointModel(t *testing.T) {
	store := newTestSkillToolStorage(t)
	saveTestProvider(t, store, &storage.Provider{
		Name:         "qianfan",
		Type:         consts.ProviderAnthropic,
		Protocol:     consts.ProtocolAnthropic,
		APIBase:      "https://qianfan.baidubce.com/anthropic/coding",
		APIKey:       "test-key",
		DefaultModel: "glm-5",
		Enabled:      true,
	})
	saveTestProvider(t, store, &storage.Provider{
		Name:         "openai-main",
		Type:         consts.ProviderOpenAI,
		Protocol:     consts.ProtocolOpenAI,
		APIBase:      "https://api.openai.com/v1",
		APIKey:       "test-key",
		DefaultModel: "gpt-4o-mini",
		Enabled:      true,
	})
	saveTestParam(t, store, consts.DEFAULT_MODEL_KEY, "qianfan/glm-5")

	manager := providers.NewManager(store, nil)
	tool := NewExecuteSkillTool("/workspace", store, manager, nil, nil)

	provider, modelName, err := tool.resolveProvider()
	if err != nil {
		t.Fatalf("resolveProvider() error = %v", err)
	}
	if provider.GetName() != "anthropic" {
		t.Fatalf("provider name = %s, want anthropic", provider.GetName())
	}
	if modelName != "glm-5" {
		t.Fatalf("model name = %s, want glm-5", modelName)
	}
}

func TestExecuteSkillTool_ResolveProvider_FallbackSkipsUnsupportedCodingModel(t *testing.T) {
	store := newTestSkillToolStorage(t)
	saveTestProvider(t, store, &storage.Provider{
		Name:         "qianfan",
		Type:         consts.ProviderAnthropic,
		Protocol:     consts.ProtocolAnthropic,
		APIBase:      "https://qianfan.baidubce.com/anthropic/coding",
		APIKey:       "test-key",
		DefaultModel: "unsupported-model",
		LLMs: storage.LLMs{
			{Alias: "unsupported", Model: "unsupported-model"},
			{Alias: "glm", Model: "glm-5"},
		},
		Enabled: true,
	})
	saveTestParam(t, store, consts.DEFAULT_MODEL_KEY, "qianfan/glm-5")

	manager := providers.NewManager(store, nil)
	tool := NewExecuteSkillTool("/workspace", store, manager, nil, nil)

	provider, modelName, err := tool.resolveProvider()
	if err != nil {
		t.Fatalf("resolveProvider() error = %v", err)
	}
	if provider.GetName() != "anthropic" {
		t.Fatalf("provider name = %s, want anthropic", provider.GetName())
	}
	if modelName != "glm-5" {
		t.Fatalf("model name = %s, want glm-5", modelName)
	}
}

func TestExecuteSkillTool_ResolveProvider_ReturnsClearErrorWhenConfiguredCodingModelUnsupported(t *testing.T) {
	store := newTestSkillToolStorage(t)
	saveTestProvider(t, store, &storage.Provider{
		Name:         "qianfan",
		Type:         consts.ProviderAnthropic,
		Protocol:     consts.ProtocolAnthropic,
		APIBase:      "https://qianfan.baidubce.com/anthropic/coding",
		APIKey:       "test-key",
		DefaultModel: "unsupported-model",
		Enabled:      true,
	})
	saveTestParam(t, store, consts.SKILL_DEFAULT_MODEL_KEY, "qianfan/unsupported-model")

	manager := providers.NewManager(store, nil)
	tool := NewExecuteSkillTool("/workspace", store, manager, nil, nil)

	_, _, err := tool.resolveProvider()
	if err == nil {
		t.Fatal("resolveProvider() error = nil, want error")
	}
	if !strings.Contains(err.Error(), consts.SKILL_DEFAULT_MODEL_KEY) {
		t.Fatalf("resolveProvider() error = %v, want mention %s", err, consts.SKILL_DEFAULT_MODEL_KEY)
	}
	if !strings.Contains(err.Error(), "unsupported-model") {
		t.Fatalf("resolveProvider() error = %v, want model detail", err)
	}
}

func newTestSkillToolStorage(t *testing.T) *storage.Storage {
	t.Helper()

	workspace := t.TempDir()
	store, err := storage.New(workspace, "", filepath.Join(workspace, "skill-tool.db"))
	if err != nil {
		t.Fatalf("storage.New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("store.Close() error = %v", err)
		}
	})
	return store
}

func saveTestProvider(t *testing.T, store *storage.Storage, provider *storage.Provider) {
	t.Helper()
	if err := store.Provider().Save(provider); err != nil {
		t.Fatalf("Provider().Save() error = %v", err)
	}
}

func saveTestParam(t *testing.T, store *storage.Storage, key string, value string) {
	t.Helper()
	if err := store.Param().SaveOrUpdateByKey(&storage.ParamConfig{
		Key:         key,
		Value:       value,
		Description: key,
		Group:       "test",
		Enabled:     true,
	}); err != nil {
		t.Fatalf("Param().SaveOrUpdateByKey() error = %v", err)
	}
}
