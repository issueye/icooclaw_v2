package skill

import (
	"os"
	"path/filepath"
	"testing"

	"icooclaw/pkg/storage"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newSyncTestSkillStorage(t *testing.T) *storage.SkillStorage {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "skills.db")
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
	if err := db.AutoMigrate(&storage.Skill{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return storage.NewSkillStorage(db)
}

func TestSyncWorkspaceSkills(t *testing.T) {
	store := newSyncTestSkillStorage(t)
	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, "skills", "amap-weather-1.0.0")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	content := `---
name: AMap Weather
slug: amap-weather
description: Weather sync test
version: 1.0.0
---

# Skill Body`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	invalidDir := filepath.Join(workspace, "skills", "broken-skill")
	if err := os.MkdirAll(invalidDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() invalid error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(invalidDir, "SKILL.md"), []byte("# no frontmatter"), 0o600); err != nil {
		t.Fatalf("WriteFile() invalid error = %v", err)
	}

	if err := SyncWorkspaceSkills(workspace, store); err != nil {
		t.Fatalf("SyncWorkspaceSkills() error = %v", err)
	}

	skill, err := store.GetSkill("amap-weather")
	if err != nil {
		t.Fatalf("GetSkill() error = %v", err)
	}
	if skill.Title != "AMap Weather" {
		t.Fatalf("Title = %q, want AMap Weather", skill.Title)
	}
	if skill.Path != "skills/amap-weather-1.0.0" {
		t.Fatalf("Path = %q, want skills/amap-weather-1.0.0", skill.Path)
	}

	skills, err := store.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills() error = %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("ListSkills() len = %d, want 1", len(skills))
	}
}
