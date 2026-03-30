package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ensureWorkDir(workDir string) string {
	if workDir == "" {
		workDir = "./workspace"
	}
	_ = os.MkdirAll(workDir, 0o755)
	return workDir
}

func resolvePath(workDir, path string) (string, error) {
	base := ensureWorkDir(workDir)
	target := filepath.Join(base, filepath.Clean(path))

	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("获取工作目录绝对路径失败: %w", err)
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("获取目标路径绝对路径失败: %w", err)
	}

	if !strings.HasPrefix(absTarget, absBase) {
		return "", fmt.Errorf("路径超出工作目录范围")
	}

	return absTarget, nil
}
