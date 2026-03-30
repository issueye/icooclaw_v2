package storage

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"icooclaw/pkg/consts"
)

func newTestProviderStorage(t *testing.T) *ProviderStorage {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "providers.db")
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
	if err := db.AutoMigrate(&Provider{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return NewProviderStorage(db)
}

func TestProviderStorage_Save_InferProtocolFromType(t *testing.T) {
	store := newTestProviderStorage(t)

	provider := &Provider{
		Name:         "anthropic-main",
		Type:         consts.ProviderAnthropic,
		APIKey:       "test-key",
		DefaultModel: "claude-3-5-sonnet-20241022",
	}
	if err := store.Save(provider); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if provider.Protocol != consts.ProtocolAnthropic {
		t.Fatalf("provider protocol = %s, want %s", provider.Protocol, consts.ProtocolAnthropic)
	}
}

func TestProviderStorage_Save_InferMiniMaxProtocolFromLegacyConfig(t *testing.T) {
	store := newTestProviderStorage(t)

	provider := &Provider{
		Name:         "minimax-openai",
		Type:         consts.ProviderMiniMax,
		APIKey:       "test-key",
		DefaultModel: "MiniMax-M2.5",
		Config:       `{"api_format":"openai"}`,
	}
	if err := store.Save(provider); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if provider.Protocol != consts.ProtocolOpenAI {
		t.Fatalf("provider protocol = %s, want %s", provider.Protocol, consts.ProtocolOpenAI)
	}
}

func TestProviderStorage_GetByName_BackfillsMissingProtocol(t *testing.T) {
	store := newTestProviderStorage(t)

	legacy := &Provider{
		Name:         "legacy-anthropic",
		Type:         consts.ProviderAnthropic,
		Protocol:     "",
		APIKey:       "test-key",
		DefaultModel: "claude-3-5-sonnet-20241022",
	}
	if err := store.db.Create(legacy).Error; err != nil {
		t.Fatalf("create legacy provider: %v", err)
	}

	loaded, err := store.GetByName("legacy-anthropic")
	if err != nil {
		t.Fatalf("GetByName() error = %v", err)
	}
	if loaded.Protocol != consts.ProtocolAnthropic {
		t.Fatalf("loaded protocol = %s, want %s", loaded.Protocol, consts.ProtocolAnthropic)
	}

	var persisted Provider
	if err := store.db.Where("name = ?", "legacy-anthropic").First(&persisted).Error; err != nil {
		t.Fatalf("reload provider: %v", err)
	}
	if persisted.Protocol != consts.ProtocolAnthropic {
		t.Fatalf("persisted protocol = %s, want %s", persisted.Protocol, consts.ProtocolAnthropic)
	}
}
