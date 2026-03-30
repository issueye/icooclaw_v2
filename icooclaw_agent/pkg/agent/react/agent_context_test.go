package react

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/skill"
	"icooclaw/pkg/storage"
)

type mockSkillLoader struct{}

func (m mockSkillLoader) LoadMetadata(ctx context.Context, name string) (*skill.Metadata, error) {
	return nil, nil
}

func (m mockSkillLoader) LoadInfo(ctx context.Context, name string) (*skill.Info, error) {
	return nil, nil
}

func (m mockSkillLoader) List(ctx context.Context) ([]*skill.Info, error) {
	return []*skill.Info{}, nil
}

func TestBuildMessages_AppendsAgentListAndCurrentAgent(t *testing.T) {
	workspace := t.TempDir()
	writeFile := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(workspace, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	writeFile("SOUL.md", "# Soul\nbase")
	writeFile("USER.md", "# User\nbase")

	store, err := storage.New(workspace, "", filepath.Join(workspace, "test.db"))
	if err != nil {
		t.Fatalf("storage.New() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	agentInfo := &storage.Agent{
		Name:         "researcher",
		Description:  "deep research agent",
		SystemPrompt: "focus on evidence",
		Enabled:      true,
	}
	if err := store.Agent().Save(agentInfo); err != nil {
		t.Fatalf("Save() agent error = %v", err)
	}
	agentInfo, err = store.Agent().GetByName("researcher")
	if err != nil {
		t.Fatalf("GetByName() error = %v", err)
	}

	a := &ReActAgent{
		storage: store,
		skills:  mockSkillLoader{},
	}

	messages, err := a.buildMessages(context.Background(), "web:s1", bus.InboundMessage{
		Channel:   "websocket",
		SessionID: "s1",
		Text:      "hello",
		Metadata: map[string]any{
			agentIDMetadataKey: agentInfo.ID,
		},
	})
	if err != nil {
		t.Fatalf("buildMessages() error = %v", err)
	}
	if len(messages) == 0 {
		t.Fatal("expected messages")
	}
	systemPrompt := messages[0].Content
	if !strings.Contains(systemPrompt, "当前可用 Agent 列表") {
		t.Fatalf("expected agent list in system prompt: %s", systemPrompt)
	}
	if !strings.Contains(systemPrompt, agentInfo.Name) || !strings.Contains(systemPrompt, agentInfo.ID) {
		t.Fatalf("expected selected agent name/id in system prompt: %s", systemPrompt)
	}
	if !strings.Contains(systemPrompt, agentInfo.SystemPrompt) {
		t.Fatalf("expected selected agent prompt in system prompt: %s", systemPrompt)
	}
}
