package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"icooclaw/pkg/consts"

	"github.com/google/uuid"
)

type fileMessageStore struct {
	mu          sync.RWMutex
	indexPath   string
	indexDir    string
	markdownDir string
	messages    []*Message
}

func NewJSONMessageStore(path string) (MessageStore, error) {
	return newFileMessageStore(path, "")
}

func NewMarkdownMessageStore(indexPath string, dir string) (MessageStore, error) {
	return newFileMessageStore(indexPath, dir)
}

func newFileMessageStore(indexPath string, markdownDir string) (*fileMessageStore, error) {
	if strings.TrimSpace(indexPath) == "" {
		return nil, fmt.Errorf("message index path is required")
	}

	store := &fileMessageStore{
		indexPath:   indexPath,
		indexDir:    sessionIndexDir(indexPath),
		markdownDir: strings.TrimSpace(markdownDir),
		messages:    make([]*Message, 0),
	}

	if err := os.MkdirAll(store.indexDir, 0o755); err != nil {
		return nil, fmt.Errorf("create message index dir failed: %w", err)
	}
	if store.markdownDir != "" {
		if err := os.MkdirAll(store.markdownDir, 0o755); err != nil {
			return nil, fmt.Errorf("create message markdown dir failed: %w", err)
		}
	}
	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *fileMessageStore) Save(m *Message) error {
	if m == nil {
		return fmt.Errorf("message is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cp := cloneMessage(m)
	if strings.TrimSpace(cp.ID) == "" {
		cp.ID = uuid.NewString()
	}
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = now
	}
	cp.UpdatedAt = now

	for _, existing := range s.messages {
		if existing.ID == cp.ID {
			return fmt.Errorf("message already exists")
		}
	}
	s.messages = append(s.messages, cp)

	if err := s.persistLocked(); err != nil {
		return err
	}
	if err := s.writeMarkdownLocked(cp); err != nil {
		return err
	}

	return nil
}

func (s *fileMessageStore) Get(sessionID string, limit int) ([]*Message, error) {
	if limit <= 0 {
		limit = 100
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]*Message, 0)
	for _, msg := range s.messages {
		if msg.SessionID != sessionID {
			continue
		}
		if isSummaryMessage(msg.Metadata) {
			continue
		}
		filtered = append(filtered, cloneMessage(msg))
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

func (s *fileMessageStore) ListSince(sessionID string, since *time.Time) ([]*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]*Message, 0)
	for _, msg := range s.messages {
		if msg.SessionID != sessionID {
			continue
		}
		if isSummaryMessage(msg.Metadata) {
			continue
		}
		if since != nil && !msg.CreatedAt.After(*since) {
			continue
		}
		filtered = append(filtered, cloneMessage(msg))
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})
	return filtered, nil
}

func (s *fileMessageStore) GetByID(id string) (*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, msg := range s.messages {
		if msg.ID == id {
			return cloneMessage(msg), nil
		}
	}
	return nil, fmt.Errorf("message not found")
}

func (s *fileMessageStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, msg := range s.messages {
		if msg.ID != id {
			continue
		}
		s.messages = append(s.messages[:i], s.messages[i+1:]...)
		if err := s.persistLocked(); err != nil {
			return err
		}
		if s.markdownDir != "" {
			_ = os.Remove(s.markdownPath(msg))
		}
		return nil
	}
	return nil
}

func (s *fileMessageStore) Page(query *QueryMessage) (*ResQueryMessage, error) {
	if query == nil {
		query = &QueryMessage{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]Message, 0)
	for _, msg := range s.messages {
		if query.SessionID != "" && msg.SessionID != query.SessionID {
			continue
		}
		if query.Role != "" && string(msg.Role) != query.Role {
			continue
		}
		if query.KeyWord != "" && !strings.Contains(msg.Content, query.KeyWord) && !strings.Contains(msg.Thinking, query.KeyWord) {
			continue
		}
		filtered = append(filtered, *cloneMessage(msg))
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})

	res := &ResQueryMessage{}
	res.Page.Total = int64(len(filtered))
	if query.Page.Page <= 0 || query.Page.Size <= 0 {
		res.Records = filtered
		return res, nil
	}
	start := (query.Page.Page - 1) * query.Page.Size
	if start >= len(filtered) {
		res.Records = []Message{}
		return res, nil
	}
	end := start + query.Page.Size
	if end > len(filtered) {
		end = len(filtered)
	}
	res.Records = filtered[start:end]
	return res, nil
}

func (s *fileMessageStore) Search(query *SearchMessageQuery) ([]*Message, error) {
	if query == nil {
		query = &SearchMessageQuery{}
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]*Message, 0, limit)
	for _, msg := range s.messages {
		if !query.IncludeSummary && isSummaryMessage(msg.Metadata) {
			continue
		}
		if query.SessionID != "" && msg.SessionID != query.SessionID {
			continue
		}
		if query.Role != "" && string(msg.Role) != query.Role {
			continue
		}
		if query.KeyWord != "" && !strings.Contains(msg.Content, query.KeyWord) && !strings.Contains(msg.Thinking, query.KeyWord) {
			continue
		}
		if query.Since != nil && msg.CreatedAt.Before(*query.Since) {
			continue
		}
		if query.Until != nil && msg.CreatedAt.After(*query.Until) {
			continue
		}
		filtered = append(filtered, cloneMessage(msg))
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

func (s *fileMessageStore) GetSummary(sessionID string) ([]*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]*Message, 0)
	for _, msg := range s.messages {
		if msg.SessionID == sessionID && isSummaryMessage(msg.Metadata) {
			filtered = append(filtered, cloneMessage(msg))
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})
	return filtered, nil
}

func (s *fileMessageStore) GetRecentSummary(sessionID string, limit int) ([]*Message, error) {
	if limit <= 0 {
		limit = 3
	}
	summaries, err := s.GetSummary(sessionID)
	if err != nil {
		return nil, err
	}
	if len(summaries) <= limit {
		return summaries, nil
	}
	return summaries[len(summaries)-limit:], nil
}

func (s *fileMessageStore) UpdateSummary(id string, summary string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, msg := range s.messages {
		if msg.ID != id {
			continue
		}
		msg.Summary = summary
		msg.UpdatedAt = time.Now()
		if err := s.persistLocked(); err != nil {
			return err
		}
		return s.writeMarkdownLocked(msg)
	}
	return fmt.Errorf("message not found")
}

func (s *fileMessageStore) GetLastUserMessage(sessionID string) (*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var latest *Message
	for _, msg := range s.messages {
		if msg.SessionID != sessionID || msg.Role != consts.RoleUser {
			continue
		}
		if latest == nil || msg.CreatedAt.After(latest.CreatedAt) {
			latest = msg
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("no user message found")
	}
	return cloneMessage(latest), nil
}

func (s *fileMessageStore) load() error {
	entries, err := os.ReadDir(s.indexDir)
	if err != nil {
		if os.IsNotExist(err) {
			s.messages = make([]*Message, 0)
			return nil
		}
		return fmt.Errorf("read message index dir failed: %w", err)
	}

	s.messages = make([]*Message, 0)
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.indexDir, entry.Name()))
		if err != nil {
			return fmt.Errorf("read session message index failed: %w", err)
		}
		if len(data) == 0 {
			continue
		}

		var records []Message
		if err := json.Unmarshal(data, &records); err != nil {
			return fmt.Errorf("parse session message index failed: %w", err)
		}

		for i := range records {
			msg := records[i]
			s.messages = append(s.messages, cloneMessage(&msg))
		}
	}
	return nil
}

func (s *fileMessageStore) persistLocked() error {
	sessionRecords := make(map[string][]Message)
	for _, msg := range s.messages {
		sessionID := sanitizeName(msg.SessionID)
		sessionRecords[sessionID] = append(sessionRecords[sessionID], *cloneMessage(msg))
	}

	entries, err := os.ReadDir(s.indexDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read message index dir failed: %w", err)
	}
	existing := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		existing[entry.Name()] = struct{}{}
	}

	for sessionID, records := range sessionRecords {
		sort.Slice(records, func(i, j int) bool {
			return records[i].CreatedAt.Before(records[j].CreatedAt)
		})
		data, err := json.MarshalIndent(records, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal session message index failed: %w", err)
		}
		name := sessionID + ".json"
		if err := os.WriteFile(filepath.Join(s.indexDir, name), data, 0o644); err != nil {
			return fmt.Errorf("write session message index failed: %w", err)
		}
		delete(existing, name)
	}

	for name := range existing {
		if err := os.Remove(filepath.Join(s.indexDir, name)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove stale session message index failed: %w", err)
		}
	}
	return nil
}

func (s *fileMessageStore) writeMarkdownLocked(msg *Message) error {
	if s.markdownDir == "" || msg == nil {
		return nil
	}
	sessionDir := filepath.Join(s.markdownDir, sanitizeName(msg.SessionID))
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return fmt.Errorf("create markdown session dir failed: %w", err)
	}
	content := formatMessageMarkdown(msg)
	if err := os.WriteFile(s.markdownPath(msg), []byte(content), 0o644); err != nil {
		return fmt.Errorf("write markdown message failed: %w", err)
	}
	return nil
}

func (s *fileMessageStore) markdownPath(msg *Message) string {
	sessionDir := filepath.Join(s.markdownDir, sanitizeName(msg.SessionID))
	return filepath.Join(sessionDir, sanitizeName(msg.ID)+".md")
}

func sessionIndexDir(indexPath string) string {
	indexPath = strings.TrimSpace(indexPath)
	if indexPath == "" {
		return ""
	}
	ext := filepath.Ext(indexPath)
	if ext == "" {
		return indexPath
	}
	base := strings.TrimSuffix(filepath.Base(indexPath), ext)
	return filepath.Join(filepath.Dir(indexPath), base)
}

func sanitizeName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.ReplaceAll(value, ":", "_")
	if value == "" {
		return "unknown"
	}
	return value
}

func formatMessageMarkdown(msg *Message) string {
	var b strings.Builder
	b.WriteString("# Message\n\n")
	b.WriteString(fmt.Sprintf("- id: %s\n", msg.ID))
	b.WriteString(fmt.Sprintf("- session_id: %s\n", msg.SessionID))
	b.WriteString(fmt.Sprintf("- role: %s\n", msg.Role))
	b.WriteString(fmt.Sprintf("- created_at: %s\n", msg.CreatedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- updated_at: %s\n", msg.UpdatedAt.Format(time.RFC3339)))
	if msg.TotalTokens > 0 {
		b.WriteString(fmt.Sprintf("- total_tokens: %d\n", msg.TotalTokens))
	}
	b.WriteString("\n## Content\n\n")
	b.WriteString(msg.Content)
	b.WriteString("\n")
	if strings.TrimSpace(msg.Thinking) != "" {
		b.WriteString("\n## Thinking\n\n")
		b.WriteString(msg.Thinking)
		b.WriteString("\n")
	}
	if strings.TrimSpace(msg.Summary) != "" {
		b.WriteString("\n## Summary\n\n")
		b.WriteString(msg.Summary)
		b.WriteString("\n")
	}
	return b.String()
}

func cloneMessage(msg *Message) *Message {
	if msg == nil {
		return nil
	}
	cp := *msg
	return &cp
}
