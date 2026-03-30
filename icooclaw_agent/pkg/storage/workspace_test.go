package storage

import (
	"path/filepath"
	"testing"
)

func TestNormalizeWorkspacePromptName(t *testing.T) {
	for _, item := range []struct {
		input string
		want  string
	}{
		{input: "soul", want: WorkspacePromptSoul},
		{input: "USER", want: WorkspacePromptUser},
		{input: " agents ", want: WorkspacePromptAgents},
	} {
		got, err := NormalizeWorkspacePromptName(item.input)
		if err != nil {
			t.Fatalf("NormalizeWorkspacePromptName(%q) error = %v", item.input, err)
		}
		if got != item.want {
			t.Fatalf("NormalizeWorkspacePromptName(%q) = %q, want %q", item.input, got, item.want)
		}
	}

	if _, err := NormalizeWorkspacePromptName("invalid"); err == nil {
		t.Fatal("expected invalid prompt name to fail")
	}
}

func TestWorkspaceStorage_SaveAndLoad(t *testing.T) {
	workspace := t.TempDir()
	store := NewWorkspaceStorage(workspace)

	content := "# Soul\n保持沉稳。"
	if err := store.Save(WorkspacePromptSoul, content); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load(WorkspacePromptSoul)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded != content {
		t.Fatalf("Load() = %q, want %q", loaded, content)
	}

	if _, err := filepath.Abs(filepath.Join(workspace, "SOUL.md")); err != nil {
		t.Fatalf("abs path error = %v", err)
	}
}
