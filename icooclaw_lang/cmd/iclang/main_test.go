package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func TestBuildBundleProducesRunnableExecutable(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "bundle.is")
	script := `
print("hello:" + os.arg(0))
print(os.script_path())
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	outputPath := filepath.Join(dir, "bundle_app")
	if runtime.GOOS == "windows" {
		outputPath += ".exe"
	}

	runtimePath, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve test executable: %v", err)
	}

	if err := buildBundleWithRuntime(scriptPath, outputPath, runtimePath); err != nil {
		t.Fatalf("build bundle: %v", err)
	}

	cmd := exec.Command(outputPath, "-test.run=TestBundledHelperProcess", "--", "world")
	cmd.Env = append(os.Environ(), "ICLANG_BUNDLE_HELPER=1")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("run bundled executable: %v, output=%s", err, stdout.String())
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 output lines, got %d: %q", len(lines), stdout.String())
	}
	if lines[0] != "hello:world" {
		t.Fatalf("unexpected bundled output: %q", lines[0])
	}
	if lines[1] != filepath.Base(scriptPath) {
		t.Fatalf("unexpected bundled script path: %q", lines[1])
	}
}

func TestBundledHelperProcess(t *testing.T) {
	if os.Getenv("ICLANG_BUNDLE_HELPER") != "1" {
		t.SkipNow()
	}

	args := os.Args[1:]
	for i, arg := range os.Args {
		if arg == "--" {
			args = os.Args[i+1:]
			break
		}
	}

	handled, err := tryRunBundledExecutable(args)
	if err != nil {
		t.Fatalf("run bundled executable: %v", err)
	}
	if !handled {
		t.Fatal("expected bundled payload to be handled")
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
