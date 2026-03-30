package evaluator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/issueye/icooclaw_lang/internal/object"
)

func TestLogLibraryTextOutputAndLevelFiltering(t *testing.T) {
	logPath := filepath.ToSlash(filepath.Join(t.TempDir(), "script.log"))

	env, result := evalSource(t, fmt.Sprintf(`
log.reset()
log.set_output("%s")
log.set_level("warn")
log.debug("hidden debug")
log.info("hidden info")
log.warn("warn message")
log.error({"request_id": "req-1", "code": 500}, "boom")
level_name = log.level()
output_name = log.output()
json_mode = log.is_json()
log.reset()
`, logPath))

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	if got := testStringValue(t, env, "level_name"); got != "WARN" {
		t.Fatalf("expected level_name=WARN, got %s", got)
	}
	if got := testStringValue(t, env, "output_name"); got != logPath {
		t.Fatalf("expected output_name=%s, got %s", logPath, got)
	}
	if got := testStringValue(t, env, "json_mode"); got != "false" {
		t.Fatalf("expected json_mode=false, got %s", got)
	}

	data, err := os.ReadFile(filepath.FromSlash(logPath))
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "hidden debug") || strings.Contains(text, "hidden info") {
		t.Fatalf("unexpected filtered logs in file: %s", text)
	}
	if !strings.Contains(text, "WARN warn message") {
		t.Fatalf("expected warn message in log file: %s", text)
	}
	if !strings.Contains(text, "ERROR boom") {
		t.Fatalf("expected error message in log file: %s", text)
	}
	if !strings.Contains(text, "code=500") || !strings.Contains(text, "request_id=req-1") {
		t.Fatalf("expected structured fields in log file: %s", text)
	}
}

func TestLogLibraryJSONOutput(t *testing.T) {
	logPath := filepath.ToSlash(filepath.Join(t.TempDir(), "script.json.log"))

	_, result := evalSource(t, fmt.Sprintf(`
log.reset()
log.set_output("%s")
log.set_json(true)
log.set_level("debug")
log.info({"request_id": "req-2", "ok": true}, "json message")
log.reset()
`, logPath))

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	data, err := os.ReadFile(filepath.FromSlash(logPath))
	if err != nil {
		t.Fatalf("read json log file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 json log line, got %d", len(lines))
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &payload); err != nil {
		t.Fatalf("decode json log line: %v", err)
	}

	if payload["level"] != "INFO" {
		t.Fatalf("expected level=INFO, got %v", payload["level"])
	}
	if payload["message"] != "json message" {
		t.Fatalf("expected message=json message, got %v", payload["message"])
	}
	fields, ok := payload["fields"].(map[string]any)
	if !ok {
		t.Fatalf("expected fields object, got %#v", payload["fields"])
	}
	if fields["request_id"] != "req-2" {
		t.Fatalf("expected request_id=req-2, got %v", fields["request_id"])
	}
	if fields["ok"] != true {
		t.Fatalf("expected ok=true, got %v", fields["ok"])
	}
	if _, ok := payload["timestamp"].(string); !ok {
		t.Fatalf("expected timestamp string, got %#v", payload["timestamp"])
	}
}
