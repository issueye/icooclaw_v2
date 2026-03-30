package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

func newMemoryTestStorage(t *testing.T) *storage.Storage {
	t.Helper()

	workspace := t.TempDir()
	for _, name := range []string{"SOUL.md", "USER.md"} {
		if err := os.WriteFile(filepath.Join(workspace, name), []byte(name), 0o600); err != nil {
			t.Fatalf("write workspace file: %v", err)
		}
	}

	dbPath := filepath.Join(t.TempDir(), "memory.db")
	store, err := storage.New(workspace, "release", dbPath)
	if err != nil {
		t.Fatalf("storage.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}

func TestDefaultLoader_Load_CombinesSummaryAndRecentMessages(t *testing.T) {
	store := newMemoryTestStorage(t)
	sessionKey := "session-1"

	if err := store.Message().Save(&storage.Message{
		SessionID: sessionKey,
		Role:      consts.RoleSystem,
		Content:   "用户想查天气，之前已确认城市。",
		Metadata:  `{"type":"summary"}`,
	}); err != nil {
		t.Fatalf("save summary: %v", err)
	}
	if err := store.Message().Save(&storage.Message{
		SessionID: sessionKey,
		Role:      consts.RoleUser,
		Content:   "第一条",
	}); err != nil {
		t.Fatalf("save first: %v", err)
	}
	if err := store.Message().Save(&storage.Message{
		SessionID: sessionKey,
		Role:      consts.RoleAssistant,
		Content:   "第二条",
	}); err != nil {
		t.Fatalf("save second: %v", err)
	}
	if err := store.Message().Save(&storage.Message{
		SessionID: sessionKey,
		Role:      consts.RoleUser,
		Content:   "第三条",
	}); err != nil {
		t.Fatalf("save third: %v", err)
	}

	loader := NewLoaderWithRecentCount(store, 100, 2, nil)
	messages, err := loader.Load(context.Background(), sessionKey)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("Load() len = %d, want 3", len(messages))
	}
	if messages[0].Role != consts.RoleSystem.ToString() {
		t.Fatalf("summary role = %q, want system", messages[0].Role)
	}
	if messages[1].Content != "第二条" {
		t.Fatalf("recent[0] = %q, want 第二条", messages[1].Content)
	}
	if messages[2].Content != "第三条" {
		t.Fatalf("recent[1] = %q, want 第三条", messages[2].Content)
	}
}
