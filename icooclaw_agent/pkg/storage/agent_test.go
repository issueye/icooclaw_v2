package storage

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"icooclaw/pkg/consts"
)

func newTestAgentStorage(t *testing.T) *AgentStorage {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "agents.db")
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
	if err := db.AutoMigrate(&Agent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return NewAgentStorage(db)
}

func TestAgentStorage_GetDefault_CreatesDefaultAgent(t *testing.T) {
	store := newTestAgentStorage(t)

	agentInfo, err := store.GetDefault()
	if err != nil {
		t.Fatalf("GetDefault() error = %v", err)
	}
	if agentInfo.Name != consts.DEFAULT_AGENT_NAME {
		t.Fatalf("default agent name = %q, want %q", agentInfo.Name, consts.DEFAULT_AGENT_NAME)
	}
	if agentInfo.Type != AgentTypeMaster {
		t.Fatalf("default agent type = %q, want %q", agentInfo.Type, AgentTypeMaster)
	}

	agents, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("List() len = %d, want 1", len(agents))
	}
}

func TestAgentStorage_Save_UpdatesByID(t *testing.T) {
	store := newTestAgentStorage(t)

	agentInfo := &Agent{
		Name:         "analyst",
		Description:  "desc-1",
		SystemPrompt: "prompt-1",
		Enabled:      true,
	}
	if err := store.Save(agentInfo); err != nil {
		t.Fatalf("Save() create error = %v", err)
	}

	loaded, err := store.GetByName("analyst")
	if err != nil {
		t.Fatalf("GetByName() error = %v", err)
	}
	loaded.Name = "researcher"
	loaded.Description = "desc-2"
	loaded.SystemPrompt = "prompt-2"
	if err := store.Save(loaded); err != nil {
		t.Fatalf("Save() update error = %v", err)
	}

	updated, err := store.GetByID(loaded.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if updated.Name != "researcher" {
		t.Fatalf("updated name = %q, want researcher", updated.Name)
	}
	if updated.SystemPrompt != "prompt-2" {
		t.Fatalf("updated prompt = %q, want prompt-2", updated.SystemPrompt)
	}
}

func TestIsValidAgentType(t *testing.T) {
	for _, value := range []string{"", AgentTypeMaster, AgentTypeSubAgent} {
		if !IsValidAgentType(value) {
			t.Fatalf("expected valid agent type: %q", value)
		}
	}

	if IsValidAgentType("worker") {
		t.Fatal("expected worker to be invalid")
	}
}
