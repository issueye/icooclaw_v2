package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunFilePassesScriptArgsToRuntime(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "cli_args.is")
	script := `
print(os.args())
print(os.arg(0))
print(os.flag("mode"))
print(os.has_flag("verbose"))
print(os.script_path())
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	output := captureStdout(t, func() {
		runFile(scriptPath, []string{"input.txt", "--mode=prod", "--verbose"})
	})

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 output lines, got %d: %q", len(lines), output)
	}
	if lines[0] != "[input.txt, --mode=prod, --verbose]" {
		t.Fatalf("unexpected args output: %q", lines[0])
	}
	if lines[1] != "input.txt" {
		t.Fatalf("unexpected arg(0) output: %q", lines[1])
	}
	if lines[2] != "prod" {
		t.Fatalf("unexpected flag(mode) output: %q", lines[2])
	}
	if lines[3] != "true" {
		t.Fatalf("unexpected has_flag(verbose) output: %q", lines[3])
	}
	if lines[4] != scriptPath {
		t.Fatalf("unexpected script_path output: %q", lines[4])
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	defer reader.Close()

	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return buf.String()
}
