package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type ShellSpec struct {
	Command string
	Args    []string
}

func DetectShell(command string) ShellSpec {
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("pwsh"); err == nil {
			return ShellSpec{
				Command: "pwsh",
				Args:    []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command", command},
			}
		}
		if _, err := exec.LookPath("powershell"); err == nil {
			return ShellSpec{
				Command: "powershell",
				Args:    []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command", command},
			}
		}
		return ShellSpec{
			Command: "cmd",
			Args:    []string{"/C", command},
		}
	}

	return ShellSpec{
		Command: "sh",
		Args:    []string{"-c", command},
	}
}

func MergeEnvWithWorkspaceBin(baseEnv []string, workDir string) []string {
	env := append([]string{}, baseEnv...)
	if workDir == "" {
		return env
	}

	binDir := filepath.Join(workDir, "bin")
	currentPath, pathKey := lookupPathEnv(env)
	mergedPath := binDir
	if currentPath != "" {
		mergedPath += string(os.PathListSeparator) + currentPath
	}

	replaced := false
	for i, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if isPathEnvKey(key) {
			env[i] = pathKey + "=" + mergedPath
			replaced = true
			break
		}
	}
	if !replaced {
		env = append(env, pathKey+"="+mergedPath)
	}

	return env
}

func lookupPathEnv(env []string) (value string, key string) {
	for _, entry := range env {
		k, v, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if isPathEnvKey(k) {
			return v, k
		}
	}

	if runtime.GOOS == "windows" {
		return os.Getenv("Path"), "Path"
	}
	return os.Getenv("PATH"), "PATH"
}

func isPathEnvKey(key string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(key, "PATH")
	}
	return key == "PATH"
}
