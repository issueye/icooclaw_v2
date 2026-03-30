package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLogger_WithFileOutput(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "logs", "app.log")

	logger, closer, err := NewLogger(Options{
		Level:  slog.LevelInfo,
		Format: "text",
		Output: logPath,
		Stdout: io.Discard,
	})
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	if closer != nil {
		defer closer.Close()
	}

	logger.Info("test message", "component", "test")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "test message") {
		t.Fatalf("log file content unexpected: %s", string(data))
	}
}
