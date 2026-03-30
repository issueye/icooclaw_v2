package storage

import (
	"os"
	"path/filepath"
	"testing"

	"icooclaw/pkg/consts"
)

func TestJSONMessageStore_SaveAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "messages.json")
	store, err := NewJSONMessageStore(indexPath)
	if err != nil {
		t.Fatalf("NewJSONMessageStore() error = %v", err)
	}

	if err := store.Save(&Message{
		SessionID: "s-json",
		Role:      consts.RoleAssistant,
		Content:   "json message",
		Thinking:  "thinking",
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	messages, err := store.Get("s-json", 10)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("Get() len = %d, want 1", len(messages))
	}
	if messages[0].Content != "json message" {
		t.Fatalf("Get() content = %q, want %q", messages[0].Content, "json message")
	}

	sessionIndexPath := filepath.Join(tmpDir, "messages", "s-json.json")
	if _, err := os.Stat(sessionIndexPath); err != nil {
		t.Fatalf("session index not created: %v", err)
	}
	if _, err := os.Stat(indexPath); !os.IsNotExist(err) {
		t.Fatalf("legacy aggregate index should not exist, err = %v", err)
	}
}

func TestMarkdownMessageStore_SaveWritesMarkdownFile(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "messages_index.json")
	markdownDir := filepath.Join(tmpDir, "md")
	store, err := NewMarkdownMessageStore(indexPath, markdownDir)
	if err != nil {
		t.Fatalf("NewMarkdownMessageStore() error = %v", err)
	}

	if err := store.Save(&Message{
		Model:     Model{ID: "msg-md-1"},
		SessionID: "s-md",
		Role:      consts.RoleUser,
		Content:   "hello markdown",
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	filePath := filepath.Join(markdownDir, "s-md", "msg-md-1.md")
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("markdown file not created: %v", err)
	}

	sessionIndexPath := filepath.Join(tmpDir, "messages_index", "s-md.json")
	if _, err := os.Stat(sessionIndexPath); err != nil {
		t.Fatalf("session index not created: %v", err)
	}
	if _, err := os.Stat(indexPath); !os.IsNotExist(err) {
		t.Fatalf("legacy aggregate index should not exist, err = %v", err)
	}
}
