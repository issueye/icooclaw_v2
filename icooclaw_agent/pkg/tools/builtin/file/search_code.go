package file

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"icooclaw/pkg/tools"
)

type SearchCodeTool struct {
	WorkDir string
}

func NewSearchCodeTool(workDir string) *SearchCodeTool {
	return &SearchCodeTool{WorkDir: ensureWorkDir(workDir)}
}

func (t *SearchCodeTool) Name() string {
	return "search_code"
}

func (t *SearchCodeTool) Description() string {
	return "在工作区内按关键词搜索代码或文本内容，返回命中文件、行号和行内容。"
}

func (t *SearchCodeTool) Parameters() map[string]any {
	return map[string]any{
		"query": map[string]any{
			"type":        "string",
			"description": "要搜索的文本关键词",
			"required":    true,
		},
		"path": map[string]any{
			"type":        "string",
			"description": "搜索范围，相对工作区路径，默认 .",
		},
		"file_pattern": map[string]any{
			"type":        "string",
			"description": "文件名匹配模式，例如 *.go、*.vue",
		},
		"case_sensitive": map[string]any{
			"type":        "boolean",
			"description": "是否区分大小写，默认 false",
		},
		"max_results": map[string]any{
			"type":        "integer",
			"description": "最大返回条数，默认 20",
		},
	}
}

func (t *SearchCodeTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	query, ok := args["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return &tools.Result{Success: false, Error: fmt.Errorf("需要提供 query 参数")}
	}

	searchPath := "."
	if value, ok := args["path"].(string); ok && strings.TrimSpace(value) != "" {
		searchPath = value
	}
	filePattern, _ := args["file_pattern"].(string)
	caseSensitive, _ := args["case_sensitive"].(bool)
	maxResults := intValue(args["max_results"], 20)
	if maxResults <= 0 {
		maxResults = 20
	}

	root, err := resolvePath(t.WorkDir, searchPath)
	if err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	matches := make([]map[string]any, 0, maxResults)
	needle := query
	if !caseSensitive {
		needle = strings.ToLower(query)
	}

	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if len(matches) >= maxResults {
			return io.EOF
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "dist" || name == "bin" {
				return filepath.SkipDir
			}
			return nil
		}
		if filePattern != "" {
			ok, err := filepath.Match(filePattern, filepath.Base(path))
			if err != nil || !ok {
				return nil
			}
		}
		if isLikelyBinary(path) {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			line := scanner.Text()
			target := line
			if !caseSensitive {
				target = strings.ToLower(line)
			}
			if !strings.Contains(target, needle) {
				continue
			}

			rel, _ := filepath.Rel(ensureWorkDir(t.WorkDir), path)
			matches = append(matches, map[string]any{
				"path": rel,
				"line": lineNo,
				"text": line,
			})
			if len(matches) >= maxResults {
				return io.EOF
			}
		}
		return nil
	})

	if walkErr != nil && walkErr != io.EOF {
		return &tools.Result{Success: false, Error: walkErr}
	}

	data, _ := json.MarshalIndent(map[string]any{
		"query":   query,
		"matches": matches,
		"count":   len(matches),
	}, "", "  ")
	return &tools.Result{Success: true, Content: string(data)}
}

func isLikelyBinary(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return true
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return true
	}
	return strings.Contains(string(buf[:n]), "\x00")
}

func intValue(value any, fallback int) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return fallback
	}
}
