package file

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchCodeToolExecute(t *testing.T) {
	workDir := t.TempDir()
	filePath := filepath.Join(workDir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result := NewSearchCodeTool(workDir).Execute(context.Background(), map[string]any{
		"query":        "println",
		"file_pattern": "*.go",
	})
	if !result.Success {
		t.Fatalf("Execute() success = false, err = %v", result.Error)
	}
	if !strings.Contains(result.Content, "\"path\": \"main.go\"") {
		t.Fatalf("unexpected result content: %s", result.Content)
	}
}

func TestReplaceInFileToolExecute(t *testing.T) {
	workDir := t.TempDir()
	path := filepath.Join(workDir, "config.txt")
	if err := os.WriteFile(path, []byte("port=8080\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result := NewReplaceInFileTool(workDir).Execute(context.Background(), map[string]any{
		"path":     "config.txt",
		"old_text": "8080",
		"new_text": "9090",
	})
	if !result.Success {
		t.Fatalf("Execute() success = false, err = %v", result.Error)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "port=9090\n" {
		t.Fatalf("content = %q, want %q", string(content), "port=9090\n")
	}
}

func TestInsertInFileToolExecute(t *testing.T) {
	workDir := t.TempDir()
	path := filepath.Join(workDir, "app.js")
	original := "function start() {\n  return true;\n}\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result := NewInsertInFileTool(workDir).Execute(context.Background(), map[string]any{
		"path":    "app.js",
		"mode":    "before",
		"anchor":  "function start() {",
		"content": "// generated helper",
	})
	if !result.Success {
		t.Fatalf("Execute() success = false, err = %v", result.Error)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.HasPrefix(string(content), "// generated helper\nfunction start() {") {
		t.Fatalf("unexpected file content: %q", string(content))
	}
}
