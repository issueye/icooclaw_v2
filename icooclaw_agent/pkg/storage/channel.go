package storage

import (
	"fmt"

	"gorm.io/gorm"

	"icooclaw/pkg/adapter"
	"icooclaw/pkg/consts"
	icooclawErrors "icooclaw/pkg/errors"
)

// Channel represents a channel configuration.
type Channel struct {
	Model
	Name        string `gorm:"column:name;type:varchar(100);not null;comment:渠道名称" json:"name"`         // 渠道名称
	Type        string `gorm:"column:type;type:varchar(50);not null;comment:渠道类型" json:"type"`          // 渠道类型
	Enabled     bool   `gorm:"column:enabled;type:tinyint(1);default:true;comment:是否启用" json:"enabled"` // 是否启用
	Config      string `gorm:"column:config;type:text;comment:配置(JSON格式)" json:"config"`                // JSON object
	Permissions string `gorm:"column:permissions;type:text;comment:权限(JSON数组)" json:"permissions"`      // JSON array
}

// TableName returns the table name for Channel.
func (Channel) TableName() string {
	return tableNamePrefix + "channels"
}

type ChannelStorage struct {
	db *gorm.DB
}

func NewChannelStorage(db *gorm.DB) *ChannelStorage {
	return &ChannelStorage{db: db}
}

// SaveChannel saves a channel configuration.
func (s *ChannelStorage) SaveChannel(c *Channel) error {
	result := s.db.Save(c)
	return result.Error
}

// GetChannel gets a channel by name.
func (s *ChannelStorage) GetChannel(name string) (*Channel, error) {
	var c Channel
	result := s.db.Where("name = ?", name).First(&c)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get channel: %w", result.Error)
	}
	return &c, nil
}

func (s *ChannelStorage) List() ([]adapter.ChannelInfo, error) {
	channels, err := s.ListEnabledChannels()
	if err != nil {
		return nil, err
	}

	resList := make([]adapter.ChannelInfo, 0, len(channels))
	for _, channel := range channels {
		resList = append(resList, adapter.ChannelInfo{
			Type:   channel.Type,
			Config: channel.Config,
		})
	}
	return resList, nil
}

// ListChannels lists all channels.
func (s *ChannelStorage) ListChannels() ([]*Channel, error) {
	var channels []*Channel
	if err := listOrdered(s.db.Model(&Channel{}), "name", &channels, "channels"); err != nil {
		return nil, err
	}
	return channels, nil
}

// ListEnabledChannels lists all enabled channels.
func (s *ChannelStorage) ListEnabledChannels() ([]*Channel, error) {
	var channels []*Channel
	if err := listOrdered(s.db.Model(&Channel{}).Where("enabled = ?", true), "name", &channels, "enabled channels"); err != nil {
		return nil, err
	}
	return channels, nil
}

// Delete deletes a channel by ID.
func (s *ChannelStorage) Delete(id string) error {
	return deleteByField(s.db, "id", id, &Channel{}, "channel")
}

// DeleteChannel deletes a channel by name.
func (s *ChannelStorage) DeleteChannel(name string) error {
	return deleteByField(s.db, "name", name, &Channel{}, "channel")
}

type QueryChannel struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
	Type    string `json:"type"`
	Enabled *bool  `json:"enabled"`
}

type ResQueryChannel struct {
	Page    Page      `json:"page"`
	Records []Channel `json:"records"`
}

// Page gets channels with pagination.
func (s *ChannelStorage) Page(query *QueryChannel) (*ResQueryChannel, error) {
	var res ResQueryChannel

	qry := s.db.Model(&Channel{})

	if query.KeyWord != "" {
		qry = qry.Where("name LIKE ? OR type LIKE ?", "%"+query.KeyWord+"%", "%"+query.KeyWord+"%")
	}

	if query.Type != "" {
		qry = qry.Where("type = ?", query.Type)
	}

	if query.Enabled != nil {
		qry = qry.Where("enabled = ?", *query.Enabled)
	}

	page, err := pageQuery(qry, "name", query.Page, &res.Records, "channels")
	if err != nil {
		return nil, err
	}
	res.Page = page

	return &res, nil
}

func (s *ChannelStorage) ExistCreate() error {
	ch := &Channel{
		Name:        consts.WEBSOCKET,
		Type:        consts.WEBSOCKET,
		Enabled:     true,
		Config:      "{}",
		Permissions: "[]",
	}

	// 检查渠道是否存在
	var existingChannel int64
	result := s.db.Model(ch).Where("name = ?", ch.Name).Count(&existingChannel)
	if result.Error != nil {
		return fmt.Errorf("failed to save channel: %w", result.Error)
	}

	if existingChannel == 0 {
		// 如果渠道不存在，则创建
		result = s.db.Save(ch)
	}

	return nil
}
