package builtin

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"icooclaw/pkg/config"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/envmgr"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
)

func newTestStorage(t *testing.T) *storage.Storage {
	t.Helper()

	workspace := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "tools.db")

	st, err := storage.New(workspace, "test", dbPath)
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})
	return st
}

func TestCreateGetUpdateDeleteAgentTools(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	createTool := NewCreateAgentTool(st)
	createResult := createTool.Execute(ctx, map[string]any{
		"name":          "researcher",
		"type":          storage.AgentTypeSubAgent,
		"description":   "handles research",
		"system_prompt": "be precise",
		"metadata": map[string]any{
			"team": "ops",
		},
	})
	if !createResult.Success || createResult.Error != nil {
		t.Fatalf("create agent failed: %v", createResult.Error)
	}
	if !strings.Contains(createResult.Content, `"name": "researcher"`) {
		t.Fatalf("unexpected create result: %s", createResult.Content)
	}
	if !strings.Contains(createResult.Content, `"type": "subagent"`) {
		t.Fatalf("expected type in create result: %s", createResult.Content)
	}

	created, err := st.Agent().GetByName("researcher")
	if err != nil {
		t.Fatalf("load created agent: %v", err)
	}

	getTool := NewGetAgentTool(st)
	getResult := getTool.Execute(ctx, map[string]any{"id": created.ID})
	if !getResult.Success || getResult.Error != nil {
		t.Fatalf("get agent failed: %v", getResult.Error)
	}
	if !strings.Contains(getResult.Content, `"description": "handles research"`) {
		t.Fatalf("unexpected get result: %s", getResult.Content)
	}

	updateTool := NewUpdateAgentTool(st)
	updateResult := updateTool.Execute(ctx, map[string]any{
		"id":            created.ID,
		"description":   "handles deep research",
		"system_prompt": "be very precise",
		"enabled":       false,
		"metadata":      `{"team":"strategy"}`,
	})
	if !updateResult.Success || updateResult.Error != nil {
		t.Fatalf("update agent failed: %v", updateResult.Error)
	}

	updated, err := st.Agent().GetByID(created.ID)
	if err != nil {
		t.Fatalf("load updated agent: %v", err)
	}
	if updated.Description != "handles deep research" {
		t.Fatalf("description = %q", updated.Description)
	}
	if updated.Enabled {
		t.Fatalf("expected agent to be disabled")
	}
	if updated.Metadata["team"] != "strategy" {
		t.Fatalf("metadata team = %#v", updated.Metadata["team"])
	}
	if updated.Type != storage.AgentTypeSubAgent {
		t.Fatalf("type = %q", updated.Type)
	}

	listTool := NewListAgentsTool(st)
	listResult := listTool.Execute(ctx, map[string]any{"enabled": false, "keyword": "deep"})
	if !listResult.Success || listResult.Error != nil {
		t.Fatalf("list agents failed: %v", listResult.Error)
	}
	if !strings.Contains(listResult.Content, `"name": "researcher"`) {
		t.Fatalf("unexpected list result: %s", listResult.Content)
	}

	deleteTool := NewDeleteAgentTool(st)
	deleteResult := deleteTool.Execute(ctx, map[string]any{"id": created.ID})
	if !deleteResult.Success || deleteResult.Error != nil {
		t.Fatalf("delete agent failed: %v", deleteResult.Error)
	}

	if _, err := st.Agent().GetByID(created.ID); err == nil {
		t.Fatal("expected deleted agent to be missing")
	}
}

func TestCreateAgentTool_RejectsInvalidType(t *testing.T) {
	st := newTestStorage(t)

	result := NewCreateAgentTool(st).Execute(context.Background(), map[string]any{
		"name": "invalid-agent",
		"type": "worker",
	})
	if result.Success || result.Error == nil {
		t.Fatal("expected invalid type to fail")
	}
}

func TestRegisterBuiltinTools_RegistersAgentCRUDTools(t *testing.T) {
	registry := tools.NewRegistry()
	st := newTestStorage(t)

	RegisterBuiltinTools(registry, st, config.DefaultConfig())

	for _, name := range []string{
		"list_agents",
		"get_agent",
		"create_agent",
		"update_agent",
		"delete_agent",
		"search_messages",
	} {
		if !registry.HasTool(name) {
			t.Fatalf("expected tool %s to be registered", name)
		}
	}
}

func TestSearchMessagesTool_SearchesThinkingContent(t *testing.T) {
	st := newTestStorage(t)

	if err := st.Message().Save(&storage.Message{
		SessionID: "session-tool-1",
		Role:      consts.RoleAssistant,
		Content:   "最终回复",
		Thinking:  "这里有检索关键字",
	}); err != nil {
		t.Fatalf("save message: %v", err)
	}

	result := NewSearchMessagesTool(st).Execute(context.Background(), map[string]any{
		"session_id": "session-tool-1",
		"keyword":    "检索关键字",
		"limit":      5,
	})
	if !result.Success || result.Error != nil {
		t.Fatalf("search messages failed: %v", result.Error)
	}
	if !strings.Contains(result.Content, `"count": 1`) {
		t.Fatalf("unexpected search result: %s", result.Content)
	}
	if !strings.Contains(result.Content, `"content": "最终回复"`) {
		t.Fatalf("expected matched message content: %s", result.Content)
	}
}

func TestResolveRuntimeExecEnv(t *testing.T) {
	st := newTestStorage(t)
	if err := st.ExecEnv().ReplaceAll(map[string]string{"DEMO_KEY": "demo-value"}); err != nil {
		t.Fatalf("save exec env: %v", err)
	}

	values := resolveRuntimeExecEnv(st)()
	if values["DEMO_KEY"] != "demo-value" {
		t.Fatalf("DEMO_KEY = %q, want demo-value", values["DEMO_KEY"])
	}
}

func TestResolveRuntimeExecEnv_FallsBackToLegacyParam(t *testing.T) {
	st := newTestStorage(t)
	if err := st.Param().SaveOrUpdateByKey(&storage.ParamConfig{
		Key:     consts.EXEC_ENV_KEY,
		Value:   envmgr.MustJSON(map[string]string{"LEGACY_KEY": "legacy-value"}),
		Group:   "exec",
		Enabled: true,
	}); err != nil {
		t.Fatalf("save legacy exec env param: %v", err)
	}

	values := resolveRuntimeExecEnv(st)()
	if values["LEGACY_KEY"] != "legacy-value" {
		t.Fatalf("LEGACY_KEY = %q, want legacy-value", values["LEGACY_KEY"])
	}
}
