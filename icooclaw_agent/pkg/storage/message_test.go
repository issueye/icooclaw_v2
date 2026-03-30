package storage

import (
	"path/filepath"
	"testing"
	"time"

	"icooclaw/pkg/consts"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestMessageStorage(t *testing.T) *MessageStorage {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "messages.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	if err := db.AutoMigrate(&Message{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return NewMessageStorage(db)
}

func TestMessageStorage_Get_ExcludesSummaryMessages(t *testing.T) {
	store := newTestMessageStorage(t)
	sessionID := "s1"

	if err := store.Save(&Message{
		SessionID: sessionID,
		Role:      consts.RoleSystem,
		Content:   "summary",
		Metadata:  `{"type":"summary"}`,
	}); err != nil {
		t.Fatalf("save summary: %v", err)
	}
	if err := store.Save(&Message{
		SessionID: sessionID,
		Role:      consts.RoleUser,
		Content:   "hello",
	}); err != nil {
		t.Fatalf("save user: %v", err)
	}

	messages, err := store.Get(sessionID, 10)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("Get() len = %d, want 1", len(messages))
	}
	if messages[0].Content != "hello" {
		t.Fatalf("Get() content = %q, want hello", messages[0].Content)
	}
}

func TestMessageStorage_ListSince_ExcludesSummaryAndHonorsCutoff(t *testing.T) {
	store := newTestMessageStorage(t)
	sessionID := "s2"

	if err := store.Save(&Message{
		SessionID: sessionID,
		Role:      consts.RoleUser,
		Content:   "first",
	}); err != nil {
		t.Fatalf("save first: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	cutoff := time.Now()
	time.Sleep(10 * time.Millisecond)
	if err := store.Save(&Message{
		SessionID: sessionID,
		Role:      consts.RoleSystem,
		Content:   "summary",
		Metadata:  `{"type":"summary"}`,
	}); err != nil {
		t.Fatalf("save summary: %v", err)
	}
	if err := store.Save(&Message{
		SessionID: sessionID,
		Role:      consts.RoleAssistant,
		Content:   "second",
	}); err != nil {
		t.Fatalf("save second: %v", err)
	}

	messages, err := store.ListSince(sessionID, &cutoff)
	if err != nil {
		t.Fatalf("ListSince() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("ListSince() len = %d, want 1", len(messages))
	}
	if messages[0].Content != "second" {
		t.Fatalf("ListSince() content = %q, want second", messages[0].Content)
	}
}

func TestMessageStorage_SaveAndGet_PreservesThinking(t *testing.T) {
	store := newTestMessageStorage(t)
	sessionID := "s3"

	if err := store.Save(&Message{
		SessionID: sessionID,
		Role:      consts.RoleAssistant,
		Content:   "最终答案",
		Thinking:  "先分析，再回答",
	}); err != nil {
		t.Fatalf("save message: %v", err)
	}

	messages, err := store.Get(sessionID, 10)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("Get() len = %d, want 1", len(messages))
	}
	if messages[0].Thinking != "先分析，再回答" {
		t.Fatalf("Get() thinking = %q, want %q", messages[0].Thinking, "先分析，再回答")
	}
}

func TestMessageStorage_Search_FiltersKeywordAndSummary(t *testing.T) {
	store := newTestMessageStorage(t)
	sessionID := "s4"

	if err := store.Save(&Message{
		SessionID: sessionID,
		Role:      consts.RoleAssistant,
		Content:   "普通回答",
		Thinking:  "包含关键推理",
	}); err != nil {
		t.Fatalf("save assistant: %v", err)
	}
	if err := store.Save(&Message{
		SessionID: sessionID,
		Role:      consts.RoleSystem,
		Content:   "摘要消息",
		Metadata:  `{"type":"summary"}`,
	}); err != nil {
		t.Fatalf("save summary: %v", err)
	}

	messages, err := store.Search(&SearchMessageQuery{
		SessionID: sessionID,
		KeyWord:   "关键推理",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("Search() len = %d, want 1", len(messages))
	}
	if messages[0].Content != "普通回答" {
		t.Fatalf("Search() content = %q, want 普通回答", messages[0].Content)
	}
}

func TestMessageStorage_Search_HonorsTimeRange(t *testing.T) {
	store := newTestMessageStorage(t)
	sessionID := "s5"

	if err := store.Save(&Message{
		SessionID: sessionID,
		Role:      consts.RoleUser,
		Content:   "old",
	}); err != nil {
		t.Fatalf("save old: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	since := time.Now()
	time.Sleep(10 * time.Millisecond)
	if err := store.Save(&Message{
		SessionID: sessionID,
		Role:      consts.RoleAssistant,
		Content:   "new",
	}); err != nil {
		t.Fatalf("save new: %v", err)
	}

	messages, err := store.Search(&SearchMessageQuery{
		SessionID: sessionID,
		Since:     &since,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("Search() len = %d, want 1", len(messages))
	}
	if messages[0].Content != "new" {
		t.Fatalf("Search() content = %q, want new", messages[0].Content)
	}
}
