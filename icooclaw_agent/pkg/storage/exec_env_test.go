package storage

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestExecEnvStorage(t *testing.T) *ExecEnvStorage {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "exec_env.db")
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
	if err := db.AutoMigrate(&ExecEnv{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return NewExecEnvStorage(db)
}

func TestExecEnvStorage_ReplaceAllAndToMap(t *testing.T) {
	store := newTestExecEnvStorage(t)

	if err := store.ReplaceAll(map[string]string{
		"DEMO_KEY": "demo-value",
		"EMPTY":    "",
	}); err != nil {
		t.Fatalf("ReplaceAll() error = %v", err)
	}

	values, err := store.ToMap()
	if err != nil {
		t.Fatalf("ToMap() error = %v", err)
	}
	if values["DEMO_KEY"] != "demo-value" {
		t.Fatalf("DEMO_KEY = %q, want demo-value", values["DEMO_KEY"])
	}
	if _, ok := values["EMPTY"]; !ok {
		t.Fatalf("expected EMPTY key to be preserved")
	}

	if err := store.ReplaceAll(map[string]string{"NEW_KEY": "new-value"}); err != nil {
		t.Fatalf("ReplaceAll() second error = %v", err)
	}

	values, err = store.ToMap()
	if err != nil {
		t.Fatalf("ToMap() second error = %v", err)
	}
	if len(values) != 1 || values["NEW_KEY"] != "new-value" {
		t.Fatalf("unexpected values after replace: %#v", values)
	}
}
