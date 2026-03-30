package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/script"

	"github.com/dop251/goja"
)

type hookScript struct {
	name    string
	relPath string
	absPath string
	source  string
	size    int64
	modTime time.Time
}

func (h *AppAgentHooks) SetWorkspace(workspace string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.workspace = workspace
	h.scriptEngine = nil
	h.hookScripts = make(map[string]hookScript)
}

func (h *AppAgentHooks) newHookScriptEngine(ctx context.Context) *script.Engine {
	cfg := script.DefaultConfig()
	cfg.Workspace = h.workspace
	cfg.AllowFileRead = false
	cfg.AllowFileWrite = false
	cfg.AllowFileDelete = false
	cfg.AllowExec = false
	cfg.AllowNetwork = false
	engine := script.NewEngineWithContext(ctx, cfg, h.logger)
	if h.storage != nil {
		engine.RegisterBuiltin(script.NewStorageBuiltin(h.storage, h.logger))
	}
	return engine
}

func (h *AppAgentHooks) loadHookScripts() error {
	path := filepath.Join(h.workspace, h.hooksDir)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if len(h.hookScripts) > 0 {
			h.logger.Info("hooks 脚本目录不存在，已清空已加载钩子", "dir", path)
		}
		h.hookScripts = make(map[string]hookScript)
		h.scriptEngine = nil
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("读取 hooks 目录失败: %w", err)
	}

	discovered := make(map[string]hookScript)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".js" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("读取 hooks 文件信息失败: %w", err)
		}
		content, err := os.ReadFile(filepath.Join(path, entry.Name()))
		if err != nil {
			return fmt.Errorf("读取 hooks 脚本失败: %w", err)
		}

		hookName := entry.Name()[:len(entry.Name())-3]
		discovered[hookName] = hookScript{
			name:    hookName,
			relPath: filepath.Join(h.hooksDir, entry.Name()),
			absPath: filepath.Join(path, entry.Name()),
			source:  string(content),
			size:    info.Size(),
			modTime: info.ModTime(),
		}
	}

	if sameHookScripts(h.hookScripts, discovered) {
		return nil
	}

	engine := h.newHookScriptEngine(context.Background())
	names := make([]string, 0, len(discovered))
	for name := range discovered {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		scriptInfo := discovered[name]
		if _, err := engine.Run(scriptInfo.source); err != nil {
			return fmt.Errorf("加载钩子脚本失败(%s): %w", scriptInfo.relPath, err)
		}
		h.logger.Info("加载钩子脚本", "hook", scriptInfo.name, "path", scriptInfo.relPath)
	}

	h.scriptEngine = nil
	h.hookScripts = discovered
	return nil
}

func sameHookScripts(current, next map[string]hookScript) bool {
	if len(current) != len(next) {
		return false
	}
	for name, script := range current {
		other, ok := next[name]
		if !ok {
			return false
		}
		if script.relPath != other.relPath || script.absPath != other.absPath || script.size != other.size || !script.modTime.Equal(other.modTime) {
			return false
		}
	}
	return true
}

// executeHook 执行钩子脚本
func (h *AppAgentHooks) executeHook(ctx context.Context, hookName string, fnName consts.HOOKType, args ...any) (any, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	_ = hookName

	if err := h.loadHookScripts(); err != nil {
		h.logger.Error("加载钩子脚本失败", "error", err)
		return nil, err
	}

	if len(h.hookScripts) == 0 {
		return nil, nil
	}

	engine := h.newHookScriptEngine(ctx)
	names := make([]string, 0, len(h.hookScripts))
	for name := range h.hookScripts {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		scriptInfo := h.hookScripts[name]
		if _, err := engine.Run(scriptInfo.source); err != nil {
			h.logger.Error("执行钩子脚本初始化失败", "hook", scriptInfo.name, "path", scriptInfo.relPath, "error", err)
			return nil, fmt.Errorf("执行钩子脚本初始化失败(%s): %w", scriptInfo.relPath, err)
		}
	}

	fn := engine.GetGlobal(fnName.ToString())
	if fn == nil {
		return nil, nil
	}

	if value, ok := fn.(goja.Value); ok {
		if goja.IsUndefined(value) || goja.IsNull(value) {
			return nil, nil
		}
	}

	result, err := engine.Call(fnName.ToString(), args...)
	if err != nil && hookName != "" {
		h.logger.Error("调用钩子函数失败", "hook", hookName, "fn", fnName.ToString(), "error", err)
		return nil, fmt.Errorf("调用钩子函数失败: %w", err)
	}
	if result == nil {
		return nil, nil
	}

	if value, ok := result.(goja.Value); ok {
		return value.Export(), nil
	}
	return result, nil
}
