package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"icooclaw/pkg/storage"
)

func newTestStorageForSkillHandler(t *testing.T) (*storage.Storage, string) {
	t.Helper()

	workspaceDir := t.TempDir()
	for _, name := range []string{"SOUL.md", "USER.md"} {
		if err := os.WriteFile(filepath.Join(workspaceDir, name), []byte("# test\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
	}

	dbPath := filepath.Join(t.TempDir(), "skills.db")
	store, err := storage.New(workspaceDir, "", dbPath)
	if err != nil {
		t.Fatalf("storage.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store, workspaceDir
}

func TestSkillHandlerImport_CreatesSkillFileAndRecord(t *testing.T) {
	store, workspaceDir := newTestStorageForSkillHandler(t)
	handler := NewSkillHandler(slog.Default(), store)

	fileContent := strings.TrimSpace(`
---
name: Demo Skill
slug: demo-skill
version: 1.2.3
description: Demo imported skill
---

Use this imported skill.
`)
	payload := map[string]any{
		"data": `{"skills":[{"name":"demo-skill","file_content":` + strconvQuote(fileContent) + `}]}`,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/skills/import", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Import(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Import() status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	skill, err := store.Skill().GetSkill("demo-skill")
	if err != nil {
		t.Fatalf("GetSkill() error = %v", err)
	}
	if skill.Version != "1.2.3" {
		t.Fatalf("skill version = %q, want %q", skill.Version, "1.2.3")
	}

	skillFile := filepath.Join(workspaceDir, "skills", "demo-skill-1.2.3", "SKILL.md")
	content, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.TrimSpace(string(content)) != fileContent {
		t.Fatalf("skill file content mismatch")
	}
}

func TestSkillHandlerImport_SkipsExistingWithoutOverwrite(t *testing.T) {
	store, workspaceDir := newTestStorageForSkillHandler(t)
	handler := NewSkillHandler(slog.Default(), store)

	existingDir := filepath.Join(workspaceDir, "skills", "demo-skill-1.0.0")
	if err := os.MkdirAll(existingDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	original := "---\nname: Demo Skill\nslug: demo-skill\ndescription: Original\nversion: 1.0.0\n---\n\noriginal"
	if err := os.WriteFile(filepath.Join(existingDir, "SKILL.md"), []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := store.Skill().SaveSkill(&storage.Skill{
		Name:        "demo-skill",
		Title:       "Demo Skill",
		Description: "Original",
		Version:     "1.0.0",
		Path:        "skills/demo-skill-1.0.0",
		Enabled:     true,
		Type:        storage.SkillTypeCustom,
	}); err != nil {
		t.Fatalf("SaveSkill() error = %v", err)
	}

	imported := "---\nname: Demo Skill\nslug: demo-skill\ndescription: Imported\nversion: 2.0.0\n---\n\nimported"
	payload := map[string]any{
		"data":      `{"skills":[{"name":"demo-skill","file_content":` + strconvQuote(imported) + `}]}`,
		"overwrite": false,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/skills/import", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Import(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Import() status = %d, want %d", rec.Code, http.StatusOK)
	}

	skill, err := store.Skill().GetSkill("demo-skill")
	if err != nil {
		t.Fatalf("GetSkill() error = %v", err)
	}
	if skill.Version != "1.0.0" {
		t.Fatalf("skill version = %q, want %q", skill.Version, "1.0.0")
	}
	if _, err := os.Stat(filepath.Join(workspaceDir, "skills", "demo-skill-2.0.0", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("unexpected imported overwrite file, err=%v", err)
	}
}

func TestSkillHandlerInstall_InstallsSkillFromRegistryZip(t *testing.T) {
	store, workspaceDir := newTestStorageForSkillHandler(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/download" {
			http.NotFound(w, r)
			return
		}

		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		file, err := zw.Create("demo-skill-1.0.0/SKILL.md")
		if err != nil {
			t.Fatalf("Create zip entry error = %v", err)
		}
		content := "---\nname: Demo Skill\nslug: demo-skill\nversion: 1.0.0\ndescription: Installed skill\n---\n\ninstalled content"
		if _, err := io.WriteString(file, content); err != nil {
			t.Fatalf("Write zip entry error = %v", err)
		}
		if err := zw.Close(); err != nil {
			t.Fatalf("Close zip error = %v", err)
		}

		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(buf.Bytes())
	}))
	defer server.Close()

	handler := NewSkillHandler(slog.Default(), store)
	handler.workspace = workspaceDir
	handler.installBaseURL = server.URL

	body, _ := json.Marshal(map[string]string{"slug": "demo-skill"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/skills/install", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Install(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Install() status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	skill, err := store.Skill().GetSkill("demo-skill")
	if err != nil {
		t.Fatalf("GetSkill() error = %v", err)
	}
	if skill.Version != "1.0.0" {
		t.Fatalf("skill version = %q, want %q", skill.Version, "1.0.0")
	}
	if _, err := os.Stat(filepath.Join(workspaceDir, "skills", "demo-skill-1.0.0", "SKILL.md")); err != nil {
		t.Fatalf("installed skill file missing: %v", err)
	}
}

func TestSkillHandlerImportZip_CreatesSkillFileAndRecord(t *testing.T) {
	store, workspaceDir := newTestStorageForSkillHandler(t)
	handler := NewSkillHandler(slog.Default(), store)

	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	file, err := zw.Create("demo-skill-1.0.0/SKILL.md")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	content := "---\nname: Demo Skill\nslug: demo-skill\nversion: 1.0.0\ndescription: Imported from zip\n---\n\nzip imported"
	if _, err := io.WriteString(file, content); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "skills.zip")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(zipBuf.Bytes()); err != nil {
		t.Fatalf("part.Write() error = %v", err)
	}
	if err := writer.WriteField("overwrite", "true"); err != nil {
		t.Fatalf("WriteField() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/skills/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	handler.Import(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Import(zip) status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	skill, err := store.Skill().GetSkill("demo-skill")
	if err != nil {
		t.Fatalf("GetSkill() error = %v", err)
	}
	if skill.Version != "1.0.0" {
		t.Fatalf("skill version = %q, want %q", skill.Version, "1.0.0")
	}
	if _, err := os.Stat(filepath.Join(workspaceDir, "skills", "demo-skill-1.0.0", "SKILL.md")); err != nil {
		t.Fatalf("imported skill file missing: %v", err)
	}
}

func strconvQuote(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}
