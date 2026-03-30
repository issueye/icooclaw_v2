package storage

import "time"

const (
	MessageStoreTypeSQLite   = "sqlite"
	MessageStoreTypeJSON     = "json"
	MessageStoreTypeMarkdown = "markdown"
)

type MessageStore interface {
	Save(m *Message) error
	Get(sessionID string, limit int) ([]*Message, error)
	ListSince(sessionID string, since *time.Time) ([]*Message, error)
	GetByID(id string) (*Message, error)
	Delete(id string) error
	Page(query *QueryMessage) (*ResQueryMessage, error)
	Search(query *SearchMessageQuery) ([]*Message, error)
	GetSummary(sessionID string) ([]*Message, error)
	GetRecentSummary(sessionID string, limit int) ([]*Message, error)
	UpdateSummary(id string, summary string) error
	GetLastUserMessage(sessionID string) (*Message, error)
}

type MessageStoreOptions struct {
	Type string
	Path string
	Dir  string
}
