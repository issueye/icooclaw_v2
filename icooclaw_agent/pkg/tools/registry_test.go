package tools

import (
	"context"
	"errors"
	icooclawerrors "icooclaw/pkg/errors"
	"testing"
)

func TestErrorResult(t *testing.T) {
	result := ErrorResult("test error message")

	if result.Success {
		t.Error("ErrorResult should have Success=false")
	}
	if result.Content != "test error message" {
		t.Errorf("Content = %q, want %q", result.Content, "test error message")
	}
	if result.Error == nil {
		t.Error("Error should not be nil")
	}
	if result.Error.Error() != "test error message" {
		t.Errorf("Error.Error() = %q, want %q", result.Error.Error(), "test error message")
	}
}

func TestSuccessResult(t *testing.T) {
	result := SuccessResult("success message")

	if !result.Success {
		t.Error("SuccessResult should have Success=true")
	}
	if result.Content != "success message" {
		t.Errorf("Content = %q, want %q", result.Content, "success message")
	}
	if result.Error != nil {
		t.Error("Error should be nil for SuccessResult")
	}
}

func TestRegistry_NewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if registry.Count() != 0 {
		t.Errorf("Count = %d, want 0", registry.Count())
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	mockTool := &mockTestTool{name: "test-tool", description: "A test tool"}
	registry.Register(mockTool)

	if registry.Count() != 1 {
		t.Errorf("Count = %d, want 1", registry.Count())
	}

	if !registry.HasTool("test-tool") {
		t.Error("HasTool should return true for registered tool")
	}
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	registry := NewRegistry()

	tool1 := &mockTestTool{name: "test-tool", description: "Tool 1"}
	tool2 := &mockTestTool{name: "test-tool", description: "Tool 2"}

	registry.Register(tool1)
	registry.Register(tool2)

	if registry.Count() != 1 {
		t.Errorf("Count = %d, want 1 (duplicate should overwrite)", registry.Count())
	}
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()

	registry.Register(&mockTestTool{name: "test-tool", description: "A test tool"})
	registry.Unregister("test-tool")

	if registry.Count() != 0 {
		t.Errorf("Count = %d, want 0", registry.Count())
	}
	if registry.HasTool("test-tool") {
		t.Error("HasTool should return false after unregister")
	}
}

func TestRegistry_Unregister_NonExistent(t *testing.T) {
	registry := NewRegistry()
	registry.Unregister("non-existent")
	if registry.Count() != 0 {
		t.Error("Unregister non-existent should not affect count")
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockTestTool{name: "test-tool", description: "A test tool"})

	tool, err := registry.Get("test-tool")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if tool.Name() != "test-tool" {
		t.Errorf("Name = %q, want %q", tool.Name(), "test-tool")
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.Get("non-existent")
	if err == nil {
		t.Error("Get non-existent should return error")
	}
	if !errors.Is(err, icooclawerrors.ErrToolNotFound) {
		t.Errorf("Error should be ErrToolNotFound, got %v", err)
	}
}

func TestRegistry_GetOK(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockTestTool{name: "test-tool", description: "A test tool"})

	t.Run("Found", func(t *testing.T) {
		tool, ok := registry.GetOK("test-tool")
		if !ok {
			t.Error("GetOK should return true for existing tool")
		}
		if tool.Name() != "test-tool" {
			t.Errorf("Name = %q, want %q", tool.Name(), "test-tool")
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		tool, ok := registry.GetOK("non-existent")
		if ok {
			t.Error("GetOK should return false for non-existent tool")
		}
		if tool != nil {
			t.Error("tool should be nil for non-existent")
		}
	})
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	registry.Register(&mockTestTool{name: "zebra-tool", description: "Zebra tool"})
	registry.Register(&mockTestTool{name: "alpha-tool", description: "Alpha tool"})
	registry.Register(&mockTestTool{name: "beta-tool", description: "Beta tool"})

	tools := registry.List()
	if len(tools) != 3 {
		t.Errorf("len(tools) = %d, want 3", len(tools))
	}

	names := registry.ListNames()
	if len(names) != 3 {
		t.Errorf("len(names) = %d, want 3", len(names))
	}

	expectedNames := []string{"alpha-tool", "beta-tool", "zebra-tool"}
	for i, name := range names {
		if name != expectedNames[i] {
			t.Errorf("names[%d] = %q, want %q", i, name, expectedNames[i])
		}
	}
}

func TestRegistry_Count(t *testing.T) {
	registry := NewRegistry()

	if registry.Count() != 0 {
		t.Errorf("initial Count = %d, want 0", registry.Count())
	}

	registry.Register(&mockTestTool{name: "tool1", description: "Tool 1"})
	if registry.Count() != 1 {
		t.Errorf("Count = %d, want 1", registry.Count())
	}

	registry.Register(&mockTestTool{name: "tool2", description: "Tool 2"})
	if registry.Count() != 2 {
		t.Errorf("Count = %d, want 2", registry.Count())
	}

	registry.Unregister("tool1")
	if registry.Count() != 1 {
		t.Errorf("Count = %d, want 1 after unregister", registry.Count())
	}
}

func TestRegistry_Execute(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockTestTool{name: "success-tool", description: "Success tool"})

	result := registry.Execute(context.Background(), "success-tool", nil)
	if !result.Success {
		t.Error("Execute should return success result")
	}
	if result.Content != "executed" {
		t.Errorf("Content = %q, want %q", result.Content, "executed")
	}
}

func TestRegistry_Execute_NotFound(t *testing.T) {
	registry := NewRegistry()

	result := registry.Execute(context.Background(), "non-existent", nil)
	if result.Success {
		t.Error("Execute should return failure for non-existent tool")
	}
}

func TestRegistry_ToProviderDefs(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockTestTool{name: "test-tool", description: "A test tool", params: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "The name",
			},
		},
	}})

	defs := registry.ToProviderDefs()
	if len(defs) != 1 {
		t.Fatalf("len(defs) = %d, want 1", len(defs))
	}

	def := defs[0]
	if def.Type != "function" {
		t.Errorf("Type = %q, want %q", def.Type, "function")
	}
	if def.Function.Name != "test-tool" {
		t.Errorf("Function.Name = %q, want %q", def.Function.Name, "test-tool")
	}
	if def.Function.Description != "A test tool" {
		t.Errorf("Function.Description = %q, want %q", def.Function.Description, "A test tool")
	}
}

func TestRegistry_GetSummaries(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockTestTool{name: "tool1", description: "Description 1"})
	registry.Register(&mockTestTool{name: "tool2", description: "Description 2"})

	summaries := registry.GetSummaries()
	if len(summaries) != 2 {
		t.Fatalf("len(summaries) = %d, want 2", len(summaries))
	}
}

func TestToolToSchema(t *testing.T) {
	tool := &mockTestTool{name: "test-tool", description: "A test tool"}
	schema := ToolToSchema(tool)

	if schema["type"] != "function" {
		t.Errorf("type = %q, want %q", schema["type"], "function")
	}

	funcDef, ok := schema["function"].(map[string]any)
	if !ok {
		t.Fatal("function should be map[string]any")
	}
	if funcDef["name"] != "test-tool" {
		t.Errorf("name = %q, want %q", funcDef["name"], "test-tool")
	}
}

func TestNormalizeToolParameters_RootRequiredStaysAtSchemaLevel(t *testing.T) {
	schema := normalizeToolParameters(map[string]any{
		"command": map[string]any{
			"type":        "string",
			"description": "要执行的命令",
		},
		"timeout": map[string]any{
			"type": "integer",
		},
		"required": []string{"command"},
	})

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties type = %T, want map[string]any", schema["properties"])
	}
	if _, exists := properties["required"]; exists {
		t.Fatalf("expected required to stay at schema root, got properties.required = %#v", properties["required"])
	}

	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatalf("required type = %T, want []string", schema["required"])
	}
	if len(required) != 1 || required[0] != "command" {
		t.Fatalf("required = %#v, want []string{\"command\"}", required)
	}
}

func TestNormalizeToolParameters_PreservesFullSchema(t *testing.T) {
	input := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type": "string",
			},
		},
		"required": []string{"name"},
	}

	schema := normalizeToolParameters(input)
	if schema["type"] != "object" {
		t.Fatalf("type = %#v, want object", schema["type"])
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok || properties["name"] == nil {
		t.Fatalf("expected full schema properties to be preserved, got %#v", schema["properties"])
	}
}

func TestNormalizeToolParameters_FieldNamedTypeDoesNotBecomeRootSchema(t *testing.T) {
	schema := normalizeToolParameters(map[string]any{
		"type": map[string]any{
			"type":        "string",
			"description": "agent type",
		},
		"name": map[string]any{
			"type": "string",
		},
	})

	if schema["type"] != "object" {
		t.Fatalf("root type = %#v, want object", schema["type"])
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties type = %T, want map[string]any", schema["properties"])
	}
	if _, ok := properties["type"].(map[string]any); !ok {
		t.Fatalf("expected field named type to stay under properties, got %#v", properties["type"])
	}
}

func TestNormalizeToolParameters_FieldNamedDescriptionDoesNotBecomeRootSchema(t *testing.T) {
	schema := normalizeToolParameters(map[string]any{
		"description": map[string]any{
			"type":        "string",
			"description": "agent description",
		},
		"name": map[string]any{
			"type": "string",
		},
	})

	if schema["type"] != "object" {
		t.Fatalf("root type = %#v, want object", schema["type"])
	}
	if _, exists := schema["description"]; exists {
		t.Fatalf("expected root description to stay unset, got %#v", schema["description"])
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties type = %T, want map[string]any", schema["properties"])
	}
	if _, ok := properties["description"].(map[string]any); !ok {
		t.Fatalf("expected field named description to stay under properties, got %#v", properties["description"])
	}
}

func TestNormalizeToolParameters_FieldLevelRequiredBooleanBecomesRootArray(t *testing.T) {
	schema := normalizeToolParameters(map[string]any{
		"destination": map[string]any{
			"type":        "string",
			"description": "目的地",
			"required":    true,
		},
		"unit": map[string]any{
			"type": "string",
		},
	})

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties type = %T, want map[string]any", schema["properties"])
	}
	destination, ok := properties["destination"].(map[string]any)
	if !ok {
		t.Fatalf("destination type = %T, want map[string]any", properties["destination"])
	}
	if _, exists := destination["required"]; exists {
		t.Fatalf("expected destination.required removed, got %#v", destination["required"])
	}

	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatalf("required type = %T, want []string", schema["required"])
	}
	if len(required) != 1 || required[0] != "destination" {
		t.Fatalf("required = %#v, want []string{\"destination\"}", required)
	}
}

type mockTestTool struct {
	name        string
	description string
	params      map[string]any
}

func (m *mockTestTool) Name() string {
	return m.name
}

func (m *mockTestTool) Description() string {
	return m.description
}

func (m *mockTestTool) Parameters() map[string]any {
	if m.params == nil {
		return map[string]any{}
	}
	return m.params
}

func (m *mockTestTool) Execute(ctx context.Context, args map[string]any) *Result {
	return SuccessResult("executed")
}
