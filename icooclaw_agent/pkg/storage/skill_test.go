package storage

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestSkillStorage(t *testing.T) *SkillStorage {
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
	if err := db.AutoMigrate(&Skill{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return NewSkillStorage(db)
}

func TestSkillStorage_GetSkill_NormalizedIdentifier(t *testing.T) {
	store := newTestSkillStorage(t)
	if err := store.SaveSkill(&Skill{
		Name:        "AMap Weather",
		Description: "legacy display-name record",
		Version:     "1.0.0",
		Enabled:     true,
		Type:        SkillTypeSkill,
		Path:        "workspace/skills/amap-weather-1.0.0",
	}); err != nil {
		t.Fatalf("SaveSkill() error = %v", err)
	}

	skill, err := store.GetSkill("amap-weather")
	if err != nil {
		t.Fatalf("GetSkill() error = %v", err)
	}
	if skill.Name != "AMap Weather" {
		t.Fatalf("GetSkill() name = %q, want legacy record", skill.Name)
	}
}

func TestSkillStorage_SaveSkill_UpsertsLegacyRecord(t *testing.T) {
	store := newTestSkillStorage(t)
	legacy := &Skill{
		Name:        "AMap Weather",
		Description: "legacy record",
		Version:     "1.0.0",
		Enabled:     true,
		Type:        SkillTypeSkill,
		Path:        "workspace/skills/amap-weather-1.0.0",
	}
	if err := store.SaveSkill(legacy); err != nil {
		t.Fatalf("SaveSkill() legacy error = %v", err)
	}

	err := store.SaveSkill(&Skill{
		Name:        "amap-weather",
		Title:       "AMap Weather",
		Description: "canonical record",
		Version:     "1.0.1",
		Enabled:     true,
		Type:        SkillTypeSkill,
		Path:        "workspace/skills/amap-weather-1.0.1",
	})
	if err != nil {
		t.Fatalf("SaveSkill() canonical error = %v", err)
	}

	skills, err := store.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills() error = %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("ListSkills() len = %d, want 1", len(skills))
	}
	if skills[0].Name != "amap-weather" {
		t.Fatalf("updated skill name = %q, want canonical slug", skills[0].Name)
	}
	if skills[0].Version != "1.0.1" {
		t.Fatalf("updated skill version = %q, want 1.0.1", skills[0].Version)
	}
}
