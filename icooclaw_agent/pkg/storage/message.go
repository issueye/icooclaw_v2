package storage

import (
	"fmt"
	"icooclaw/pkg/consts"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Message represents a chat message.
type Message struct {
	Model
	SessionID   string          `gorm:"column:session_id;type:char(36);not null;index;comment:会话ID" json:"session_id"`
	Role        consts.RoleType `gorm:"column:role;type:varchar(50);not null;comment:角色(user/assistant/system)" json:"role"`
	Content     string          `gorm:"column:content;type:text;not null;comment:消息内容" json:"content"`
	Thinking    string          `gorm:"column:thinking;type:text;comment:思考内容" json:"thinking"`
	TotalTokens int             `gorm:"column:total_tokens;type:int;default:0;comment:总token数" json:"total_tokens"`
	Summary     string          `gorm:"column:summary;type:text;comment:对话摘要" json:"summary"`
	ToolName    string          `gorm:"column:tool_name;type:varchar(50);comment:工具名称" json:"tool_name"`
	ToolArgs    string          `gorm:"column:tool_args;type:text;comment:工具参数(JSON格式)" json:"tool_args"`
	ToolResult  string          `gorm:"column:tool_result;type:text;comment:工具执行结果(JSON格式)" json:"tool_result"`
	Metadata    string          `gorm:"column:metadata;type:text;comment:元数据(JSON格式)" json:"metadata"`
}

// TableName returns the table name for Message.
func (Message) TableName() string {
	return tableNamePrefix + "messages"
}

type QueryMessage struct {
	Page      Page   `json:"page"`
	SessionID string `json:"session_id"`
	Role      string `json:"role"`
	KeyWord   string `json:"key_word"`
}

type SearchMessageQuery struct {
	SessionID      string     `json:"session_id"`
	Role           string     `json:"role"`
	KeyWord        string     `json:"key_word"`
	Since          *time.Time `json:"since"`
	Until          *time.Time `json:"until"`
	Limit          int        `json:"limit"`
	IncludeSummary bool       `json:"include_summary"`
}

type ResQueryMessage struct {
	Page    Page      `json:"page"`
	Records []Message `json:"records"`
}

type MessageStorage struct {
	db *gorm.DB
}

const summaryMessagePattern = "%\"type\":\"" + consts.SummaryMetadataType + "\"%"
const summaryMessageToken = "\"type\":\"" + consts.SummaryMetadataType + "\""

func isSummaryMessage(metadata string) bool {
	return strings.Contains(metadata, summaryMessageToken)
}

func NewMessageStorage(db *gorm.DB) *MessageStorage {
	return &MessageStorage{db: db}
}

// Save saves a message.
func (s *MessageStorage) Save(m *Message) error {
	return s.db.Create(m).Error
}

// Get gets messages by session ID.
func (s *MessageStorage) Get(sessionID string, limit int) ([]*Message, error) {
	if limit <= 0 {
		limit = 100
	}
	var messages []*Message
	result := s.db.Where("session_id = ? AND (metadata IS NULL OR metadata NOT LIKE ?)", sessionID, summaryMessagePattern).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get messages: %w", result.Error)
	}
	return messages, nil
}

// ListSince gets non-summary messages by session ID in ascending order.
func (s *MessageStorage) ListSince(sessionID string, since *time.Time) ([]*Message, error) {
	var messages []*Message

	qry := s.db.Where("session_id = ? AND (metadata IS NULL OR metadata NOT LIKE ?)", sessionID, summaryMessagePattern)
	if since != nil {
		qry = qry.Where("created_at > ?", *since)
	}

	result := qry.Order("created_at ASC").Find(&messages)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list messages: %w", result.Error)
	}

	return messages, nil
}

// GetByID gets a message by ID.
func (s *MessageStorage) GetByID(id string) (*Message, error) {
	var m Message
	result := s.db.Where("id = ?", id).First(&m)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("message not found")
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get message: %w", result.Error)
	}
	return &m, nil
}

// Delete deletes a message by ID.
func (s *MessageStorage) Delete(id string) error {
	return deleteByField(s.db, "id", id, &Message{}, "message")
}

// Page gets messages with pagination.
func (s *MessageStorage) Page(query *QueryMessage) (*ResQueryMessage, error) {
	var res ResQueryMessage

	qry := s.db.Model(&Message{})

	if query.SessionID != "" {
		qry = qry.Where("session_id = ?", query.SessionID)
	}

	if query.Role != "" {
		qry = qry.Where("role = ?", query.Role)
	}

	if query.KeyWord != "" {
		qry = qry.Where("content LIKE ? OR thinking LIKE ?", "%"+query.KeyWord+"%", "%"+query.KeyWord+"%")
	}

	page, err := pageQuery(qry, "created_at", query.Page, &res.Records, "messages")
	if err != nil {
		return nil, err
	}
	res.Page = page

	return &res, nil
}

// Search searches messages with optional session, role, keyword and time filters.
func (s *MessageStorage) Search(query *SearchMessageQuery) ([]*Message, error) {
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

	qry := s.db.Model(&Message{})
	if !query.IncludeSummary {
		qry = qry.Where("metadata IS NULL OR metadata NOT LIKE ?", summaryMessagePattern)
	}
	if query.SessionID != "" {
		qry = qry.Where("session_id = ?", query.SessionID)
	}
	if query.Role != "" {
		qry = qry.Where("role = ?", query.Role)
	}
	if query.KeyWord != "" {
		qry = qry.Where("content LIKE ? OR thinking LIKE ?", "%"+query.KeyWord+"%", "%"+query.KeyWord+"%")
	}
	if query.Since != nil {
		qry = qry.Where("created_at >= ?", *query.Since)
	}
	if query.Until != nil {
		qry = qry.Where("created_at <= ?", *query.Until)
	}

	var messages []*Message
	result := qry.Order("created_at DESC").Limit(limit).Find(&messages)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to search messages: %w", result.Error)
	}

	return messages, nil
}

// GetSummary gets summary messages by session ID.
func (s *MessageStorage) GetSummary(sessionID string) ([]*Message, error) {
	var messages []*Message
	// 查询包含 summary 类型 metadata 的消息
	result := s.db.Where("session_id = ? AND metadata LIKE ?", sessionID, summaryMessagePattern).
		Order("created_at ASC").
		Find(&messages)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get summary messages: %w", result.Error)
	}
	return messages, nil
}

// GetRecentSummary gets the latest summary messages by session ID.
func (s *MessageStorage) GetRecentSummary(sessionID string, limit int) ([]*Message, error) {
	if limit <= 0 {
		limit = 3
	}

	var messages []*Message
	result := s.db.Where("session_id = ? AND metadata LIKE ?", sessionID, summaryMessagePattern).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get recent summary messages: %w", result.Error)
	}

	// Reverse to ascending order for chronological playback.
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// UpdateSummary updates the summary field of a message.
func (s *MessageStorage) UpdateSummary(id string, summary string) error {
	result := s.db.Model(&Message{}).
		Where("id = ?", id).
		Update("summary", summary)
	if result.Error != nil {
		return fmt.Errorf("failed to update summary: %w", result.Error)
	}
	return nil
}

// GetLastUserMessage gets the last user message in a session.
func (s *MessageStorage) GetLastUserMessage(sessionID string) (*Message, error) {
	var message Message
	result := s.db.Where("session_id = ? AND role = ?", sessionID, consts.RoleUser).
		Order("created_at DESC").
		First(&message)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("no user message found")
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get last user message: %w", result.Error)
	}
	return &message, nil
}
