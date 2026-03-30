package app

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/storage"
)

func TestAppAgentHooksReloadsChangedScripts(t *testing.T) {
	workspace := t.TempDir()
	scriptPath := filepath.Join(workspace, "hooks", "hooks.js")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir hooks dir: %v", err)
	}

	writeHookScript(t, scriptPath, `function onBuildMessagesBefore(sessionKey, msg, history) {
    var messages = JSON.parse(history);
    messages.unshift({ role: "system", content: "first" });
    return { messages: messages };
}`)

	hooks := NewAppAgentHooks(HooksDependencies{
		Logger:      testHooksLogger(),
		Workspace:   workspace,
		RecentCount: consts.DefaultRecentCount,
	})
	msg := bus.InboundMessage{Channel: "web", SessionID: "s1"}
	history := []providers.ChatMessage{{Role: "user", Content: "hello"}}

	got, err := hooks.OnBuildMessagesBefore(context.Background(), "web:s1", msg, history)
	if err != nil {
		t.Fatalf("first hook execution failed: %v", err)
	}
	if len(got) != 2 || got[0].Content != "first" {
		t.Fatalf("expected first script result, got %+v", got)
	}

	writeHookScript(t, scriptPath, `function onBuildMessagesBefore(sessionKey, msg, history) {
    var messages = JSON.parse(history);
    messages.unshift({ role: "system", content: "second" });
    return { messages: messages };
}`)
	bumpFileModTime(t, scriptPath)

	got, err = hooks.OnBuildMessagesBefore(context.Background(), "web:s1", msg, history)
	if err != nil {
		t.Fatalf("reloaded hook execution failed: %v", err)
	}
	if len(got) != 2 || got[0].Content != "second" {
		t.Fatalf("expected reloaded script result, got %+v", got)
	}
}

func TestAppAgentHooksMissingFunctionFallsBack(t *testing.T) {
	workspace := t.TempDir()
	scriptPath := filepath.Join(workspace, "hooks", "custom.js")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir hooks dir: %v", err)
	}

	writeHookScript(t, scriptPath, `function onBuildMessagesBefore(sessionKey, msg, history) {
    return null;
}`)

	hooks := NewAppAgentHooks(HooksDependencies{
		Logger:      testHooksLogger(),
		Workspace:   workspace,
		RecentCount: consts.DefaultRecentCount,
	})
	msg := bus.InboundMessage{Channel: "web", SessionID: "s1"}
	history := []providers.ChatMessage{{Role: "user", Content: "hello"}}

	got, err := hooks.OnRunLLMBefore(context.Background(), msg, history)
	if err != nil {
		t.Fatalf("missing hook function should not fail: %v", err)
	}
	if len(got) != len(history) || got[0].Content != history[0].Content || got[0].Role != history[0].Role {
		t.Fatalf("expected original history fallback, got %+v", got)
	}
}

func TestAppAgentHooksIsolateVMStatePerInvocation(t *testing.T) {
	workspace := t.TempDir()
	scriptPath := filepath.Join(workspace, "hooks", "hooks.js")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir hooks dir: %v", err)
	}

	writeHookScript(t, scriptPath, `var counter = 0;
function onBuildMessagesBefore(sessionKey, msg, history) {
    counter++;
    var messages = JSON.parse(history);
    messages.unshift({ role: "system", content: String(counter) });
    return { messages: messages };
}`)

	hooks := NewAppAgentHooks(HooksDependencies{
		Logger:      testHooksLogger(),
		Workspace:   workspace,
		RecentCount: consts.DefaultRecentCount,
	})
	msg := bus.InboundMessage{Channel: "web", SessionID: "s1"}
	history := []providers.ChatMessage{{Role: "user", Content: "hello"}}

	first, err := hooks.OnBuildMessagesBefore(context.Background(), "web:s1", msg, history)
	if err != nil {
		t.Fatalf("first hook execution failed: %v", err)
	}
	second, err := hooks.OnBuildMessagesBefore(context.Background(), "web:s1", msg, history)
	if err != nil {
		t.Fatalf("second hook execution failed: %v", err)
	}

	if got := first[0].Content; got != "1" {
		t.Fatalf("expected first invocation counter 1, got %q", got)
	}
	if got := second[0].Content; got != "1" {
		t.Fatalf("expected second invocation counter 1 with fresh VM, got %q", got)
	}
}

func TestAppAgentHooksDenyNetworkAndShellByDefault(t *testing.T) {
	workspace := t.TempDir()
	scriptPath := filepath.Join(workspace, "hooks", "hooks.js")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir hooks dir: %v", err)
	}

	writeHookScript(t, scriptPath, `function onAgentStart(msg) {
    http.get("https://example.com", {});
}`)

	hooks := NewAppAgentHooks(HooksDependencies{
		Logger:      testHooksLogger(),
		Workspace:   workspace,
		RecentCount: consts.DefaultRecentCount,
	})

	_, err := hooks.executeHook(context.Background(), "hooks", consts.HookAgentStart, `{}`)
	if err == nil {
		t.Fatal("expected hook network access to be denied")
	}
	if !strings.Contains(err.Error(), "network access is not allowed") {
		t.Fatalf("expected network denial error, got %v", err)
	}

	writeHookScript(t, scriptPath, `function onAgentStart(msg) {
    shell.exec("echo hello");
}`)
	bumpFileModTime(t, scriptPath)

	_, err = hooks.executeHook(context.Background(), "hooks", consts.HookAgentStart, `{}`)
	if err == nil {
		t.Fatal("expected hook shell execution to be denied")
	}
	if !strings.Contains(err.Error(), "shell execution is not allowed") {
		t.Fatalf("expected shell denial error, got %v", err)
	}
}

func TestAppAgentHooksExposeStorageBuiltin(t *testing.T) {
	workspace := t.TempDir()
	store, err := storage.New(workspace, "", filepath.Join(workspace, "test.db"))
	if err != nil {
		t.Fatalf("storage.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	if err := store.Param().Save(&storage.ParamConfig{
		Key:         "hook.test.key",
		Value:       "hook-value",
		Description: "test param",
		Group:       "hooks",
		Enabled:     true,
	}); err != nil {
		t.Fatalf("save param: %v", err)
	}

	scriptPath := filepath.Join(workspace, "hooks", "hooks.js")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir hooks dir: %v", err)
	}

	writeHookScript(t, scriptPath, `function onAgentStart(msg) {
    var item = storage.param.get("hook.test.key");
    return { value: item.value, group: item.group };
}`)

	hooks := NewAppAgentHooks(HooksDependencies{
		Logger:      testHooksLogger(),
		Workspace:   workspace,
		Storage:     store,
		RecentCount: consts.DefaultRecentCount,
	})

	result, err := hooks.executeHook(context.Background(), "hooks", consts.HookAgentStart, `{}`)
	if err != nil {
		t.Fatalf("executeHook() error = %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if got := resultMap["value"]; got != "hook-value" {
		t.Fatalf("expected value hook-value, got %#v", got)
	}
	if got := resultMap["group"]; got != "hooks" {
		t.Fatalf("expected group hooks, got %#v", got)
	}
}

func writeHookScript(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write hook script: %v", err)
	}
}

func bumpFileModTime(t *testing.T, path string) {
	t.Helper()
	modTime := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("update script mod time: %v", err)
	}
}

func testHooksLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
