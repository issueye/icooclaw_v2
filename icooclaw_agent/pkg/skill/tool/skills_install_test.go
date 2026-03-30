package tool

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"icooclaw/pkg/storage"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newInstallTestSkillStorage(t *testing.T) *storage.SkillStorage {
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

func buildSkillZip(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := io.WriteString(w, content); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	return buf.Bytes()
}

func TestInstallSkillTool_Execute_UsesParsedVersionForPath(t *testing.T) {
	store := newInstallTestSkillStorage(t)
	workspace := t.TempDir()
	skillZip := buildSkillZip(t, map[string]string{
		"package/SKILL.md": `---
name: Demo Skill
slug: demo-skill
description: Demo install skill
version: 1.2.3
---

# Demo Skill`,
		"package/scripts/run.sh": "echo demo",
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/download" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(skillZip)
	}))
	defer server.Close()

	tool := NewInstallSkillTool(workspace, server.URL, slog.New(slog.NewTextHandler(io.Discard, nil)), store, 5*time.Second)
	result := tool.Execute(context.Background(), map[string]any{
		"slug": "demo-skill",
	})
	if !result.Success {
		t.Fatalf("Execute() success = false, error = %v", result.Error)
	}

	skill, err := store.GetSkill("demo-skill")
	if err != nil {
		t.Fatalf("GetSkill() error = %v", err)
	}
	if skill.Version != "1.2.3" {
		t.Fatalf("Version = %q, want 1.2.3", skill.Version)
	}
	if skill.Path != "skills/demo-skill-1.2.3" {
		t.Fatalf("Path = %q, want skills/demo-skill-1.2.3", skill.Path)
	}

	installedFile := filepath.Join(workspace, "skills", "demo-skill-1.2.3", "SKILL.md")
	if _, err := os.Stat(installedFile); err != nil {
		t.Fatalf("installed skill file missing: %v", err)
	}
}

func TestRemoveSkillTool_Execute_RemovesRelativeSkillPath(t *testing.T) {
	store := newInstallTestSkillStorage(t)
	workspace := t.TempDir()
	skillDir := filepath.Join(workspace, "skills", "demo-skill-1.2.3")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := store.SaveSkill(&storage.Skill{
		Name:        "demo-skill",
		Title:       "Demo Skill",
		Description: "Demo install skill",
		Version:     "1.2.3",
		Enabled:     true,
		Type:        storage.SkillTypeCustom,
		Path:        "skills/demo-skill-1.2.3",
	}); err != nil {
		t.Fatalf("SaveSkill() error = %v", err)
	}

	tool := NewRemoveSkillTool(workspace, slog.New(slog.NewTextHandler(io.Discard, nil)), store)
	result := tool.Execute(context.Background(), map[string]any{
		"slug": "demo-skill",
	})
	if !result.Success {
		t.Fatalf("Execute() success = false, error = %v", result.Error)
	}
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatalf("skill directory should be removed, stat err = %v", err)
	}
}
