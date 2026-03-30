package script

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestShellExec_ExecWithTimeout_AddsWorkspaceBinToPath(t *testing.T) {
	workspace := t.TempDir()
	binDir := filepath.Join(workspace, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	cfg := &Config{
		Workspace:   workspace,
		AllowExec:   true,
		ExecTimeout: 5,
	}
	shell := NewShellExec(context.Background(), cfg, nil)

	var command string
	expectedPrefix := binDir
	if runtime.GOOS == "windows" {
		command = "$env:Path"
		expectedPrefix = strings.ToLower(expectedPrefix)
	} else {
		command = "printf %s \"$PATH\""
	}

	result, err := shell.Exec(command)
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}

	stdout, _ := result["stdout"].(string)
	if runtime.GOOS == "windows" {
		stdout = strings.ToLower(stdout)
	}
	if !strings.Contains(stdout, expectedPrefix) {
		t.Fatalf("expected stdout to contain %q, got %q", expectedPrefix, stdout)
	}
}

func TestShellExec_Exec_InjectsConfiguredEnv(t *testing.T) {
	cfg := &Config{
		Workspace: t.TempDir(),
		AllowExec: true,
		ExecEnv: map[string]string{
			"DEMO_TOKEN": "abc123",
		},
		ExecTimeout: 5,
	}
	shell := NewShellExec(context.Background(), cfg, nil)

	var command string
	if runtime.GOOS == "windows" {
		command = "Write-Output $env:DEMO_TOKEN"
	} else {
		command = "printf %s \"$DEMO_TOKEN\""
	}

	result, err := shell.Exec(command)
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}

	stdout, _ := result["stdout"].(string)
	if !strings.Contains(stdout, "abc123") {
		t.Fatalf("expected configured env in stdout, got %q", stdout)
	}
}
