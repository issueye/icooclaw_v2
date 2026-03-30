package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"icooclaw/pkg/tools"
)

type InsertInFileTool struct {
	WorkDir string
}

func NewInsertInFileTool(workDir string) *InsertInFileTool {
	return &InsertInFileTool{WorkDir: ensureWorkDir(workDir)}
}

func (t *InsertInFileTool) Name() string {
	return "insert_in_file"
}

func (t *InsertInFileTool) Description() string {
	return "按锚点或文件首尾将文本插入文件，适合追加函数、配置块或片段。"
}

func (t *InsertInFileTool) Parameters() map[string]any {
	return map[string]any{
		"path": map[string]any{
			"type":        "string",
			"description": "目标文件路径",
			"required":    true,
		},
		"content": map[string]any{
			"type":        "string",
			"description": "要插入的内容",
			"required":    true,
		},
		"mode": map[string]any{
			"type":        "string",
			"description": "插入模式：before、after、start、end",
			"enum":        []string{"before", "after", "start", "end"},
		},
		"anchor": map[string]any{
			"type":        "string",
			"description": "before/after 模式下用于定位的锚点文本",
		},
		"ensure_newline": map[string]any{
			"type":        "boolean",
			"description": "是否为插入内容自动补换行，默认 true",
		},
	}
}

func (t *InsertInFileTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	path, ok := args["path"].(string)
	if !ok || strings.TrimSpace(path) == "" {
		return &tools.Result{Success: false, Error: fmt.Errorf("需要提供 path 参数")}
	}
	content, ok := args["content"].(string)
	if !ok || content == "" {
		return &tools.Result{Success: false, Error: fmt.Errorf("需要提供 content 参数")}
	}

	mode, _ := args["mode"].(string)
	if mode == "" {
		mode = "end"
	}
	anchor, _ := args["anchor"].(string)
	ensureNewline := true
	if value, ok := args["ensure_newline"].(bool); ok {
		ensureNewline = value
	}

	fullPath, err := resolvePath(t.WorkDir, path)
	if err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("读取文件失败: %w", err)}
	}

	insertText := content
	if ensureNewline && !strings.HasSuffix(insertText, "\n") {
		insertText += "\n"
	}

	original := string(data)
	updated, err := applyInsertion(original, insertText, mode, anchor)
	if err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("创建目录失败: %w", err)}
	}
	if err := os.WriteFile(fullPath, []byte(updated), 0o644); err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("写入文件失败: %w", err)}
	}

	return &tools.Result{
		Success: true,
		Content: fmt.Sprintf("插入完成: %s，模式 %s", path, mode),
	}
}

func applyInsertion(original, insertText, mode, anchor string) (string, error) {
	switch mode {
	case "start":
		return insertText + original, nil
	case "end":
		if original != "" && !strings.HasSuffix(original, "\n") {
			original += "\n"
		}
		return original + insertText, nil
	case "before":
		if anchor == "" {
			return "", fmt.Errorf("before 模式需要提供 anchor")
		}
		idx := strings.Index(original, anchor)
		if idx < 0 {
			return "", fmt.Errorf("未找到 anchor")
		}
		return original[:idx] + insertText + original[idx:], nil
	case "after":
		if anchor == "" {
			return "", fmt.Errorf("after 模式需要提供 anchor")
		}
		idx := strings.Index(original, anchor)
		if idx < 0 {
			return "", fmt.Errorf("未找到 anchor")
		}
		insertAt := idx + len(anchor)
		if insertAt < len(original) && original[insertAt] != '\n' && !strings.HasPrefix(insertText, "\n") {
			insertText = "\n" + insertText
		}
		return original[:insertAt] + insertText + original[insertAt:], nil
	default:
		return "", fmt.Errorf("不支持的插入模式: %s", mode)
	}
}
