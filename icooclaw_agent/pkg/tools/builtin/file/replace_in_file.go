package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"icooclaw/pkg/tools"
)

type ReplaceInFileTool struct {
	WorkDir string
}

func NewReplaceInFileTool(workDir string) *ReplaceInFileTool {
	return &ReplaceInFileTool{WorkDir: ensureWorkDir(workDir)}
}

func (t *ReplaceInFileTool) Name() string {
	return "replace_in_file"
}

func (t *ReplaceInFileTool) Description() string {
	return "在指定文件中替换文本，适合进行确定性的代码或配置修改。"
}

func (t *ReplaceInFileTool) Parameters() map[string]any {
	return map[string]any{
		"path": map[string]any{
			"type":        "string",
			"description": "要编辑的文件路径",
			"required":    true,
		},
		"old_text": map[string]any{
			"type":        "string",
			"description": "要被替换的原始文本",
			"required":    true,
		},
		"new_text": map[string]any{
			"type":        "string",
			"description": "替换后的文本",
			"required":    true,
		},
		"replace_all": map[string]any{
			"type":        "boolean",
			"description": "是否替换所有匹配，默认 false",
		},
	}
}

func (t *ReplaceInFileTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	path, ok := args["path"].(string)
	if !ok || strings.TrimSpace(path) == "" {
		return &tools.Result{Success: false, Error: fmt.Errorf("需要提供 path 参数")}
	}
	oldText, ok := args["old_text"].(string)
	if !ok || oldText == "" {
		return &tools.Result{Success: false, Error: fmt.Errorf("需要提供 old_text 参数")}
	}
	newText, ok := args["new_text"].(string)
	if !ok {
		return &tools.Result{Success: false, Error: fmt.Errorf("需要提供 new_text 参数")}
	}
	replaceAll, _ := args["replace_all"].(bool)

	fullPath, err := resolvePath(t.WorkDir, path)
	if err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("读取文件失败: %w", err)}
	}

	text := string(content)
	count := strings.Count(text, oldText)
	if count == 0 {
		return &tools.Result{Success: false, Error: fmt.Errorf("未找到要替换的文本")}
	}

	replacements := 1
	if replaceAll {
		text = strings.ReplaceAll(text, oldText, newText)
		replacements = count
	} else {
		text = strings.Replace(text, oldText, newText, 1)
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("创建目录失败: %w", err)}
	}
	if err := os.WriteFile(fullPath, []byte(text), 0o644); err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("写入文件失败: %w", err)}
	}

	return &tools.Result{
		Success: true,
		Content: fmt.Sprintf("替换完成: %s，共替换 %d 处", path, replacements),
	}
}
