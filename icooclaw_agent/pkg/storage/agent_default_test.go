package storage

import (
	"path/filepath"
	"testing"

	"icooclaw/pkg/consts"
)

func TestStorage_ResolveDefaultAgent_PrefersConfiguredID(t *testing.T) {
	workspace := t.TempDir()
	store, err := New(workspace, "", filepath.Join(workspace, "test.db"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer store.Close()

	customAgent := &Agent{
		Name:         "researcher",
		Type:         AgentTypeMaster,
		Description:  "custom default",
		SystemPrompt: "focus",
		Enabled:      true,
	}
	if err := store.Agent().Save(customAgent); err != nil {
		t.Fatalf("save custom agent: %v", err)
	}
	customAgent, err = store.Agent().GetByName("researcher")
	if err != nil {
		t.Fatalf("get custom agent: %v", err)
	}

	_ = store.Param().Delete(consts.DEFAULT_AGENT_ID_KEY)
	if err := store.Param().Save(&ParamConfig{
		Key:   consts.DEFAULT_AGENT_ID_KEY,
		Value: customAgent.ID,
	}); err != nil {
		t.Fatalf("save default agent param: %v", err)
	}

	resolved, err := store.ResolveDefaultAgent()
	if err != nil {
		t.Fatalf("ResolveDefaultAgent() error = %v", err)
	}
	if resolved == nil || resolved.ID != customAgent.ID {
		t.Fatalf("resolved default agent = %#v, want %s", resolved, customAgent.ID)
	}
}

func TestStorage_ResolveDefaultAgent_SkipsSubagentConfig(t *testing.T) {
	workspace := t.TempDir()
	store, err := New(workspace, "", filepath.Join(workspace, "test.db"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer store.Close()

	subAgent := &Agent{
		Name:         "research-subagent",
		Type:         AgentTypeSubAgent,
		Description:  "sub task executor",
		SystemPrompt: "focus",
		Enabled:      true,
	}
	if err := store.Agent().Save(subAgent); err != nil {
		t.Fatalf("save subagent: %v", err)
	}
	subAgent, err = store.Agent().GetByName("research-subagent")
	if err != nil {
		t.Fatalf("get subagent: %v", err)
	}

	_ = store.Param().Delete(consts.DEFAULT_AGENT_ID_KEY)
	if err := store.Param().Save(&ParamConfig{
		Key:   consts.DEFAULT_AGENT_ID_KEY,
		Value: subAgent.ID,
	}); err != nil {
		t.Fatalf("save default agent param: %v", err)
	}

	resolved, err := store.ResolveDefaultAgent()
	if err != nil {
		t.Fatalf("ResolveDefaultAgent() error = %v", err)
	}
	if resolved == nil || resolved.Type != AgentTypeMaster || resolved.Name != consts.DEFAULT_AGENT_NAME {
		t.Fatalf("resolved default agent = %#v, want built-in master agent", resolved)
	}
}
