package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type AgentProcessStatus struct {
	Managed       bool   `json:"managed"`
	Running       bool   `json:"running"`
	Healthy       bool   `json:"healthy"`
	PID           int    `json:"pid"`
	StartedAt     string `json:"startedAt,omitempty"`
	BinaryPath    string `json:"binaryPath,omitempty"`
	ConfigPath    string `json:"configPath,omitempty"`
	WorkingDir    string `json:"workingDir,omitempty"`
	WorkspacePath string `json:"workspacePath,omitempty"`
	APIBase       string `json:"apiBase,omitempty"`
	LastError     string `json:"lastError,omitempty"`
	LastExit      string `json:"lastExit,omitempty"`
	OutputPreview string `json:"outputPreview,omitempty"`
}

type AgentProcessManager struct {
	ctx        context.Context
	mu         sync.RWMutex
	cmd        *exec.Cmd
	startedAt  time.Time
	lastError  string
	lastExit   string
	outputBuf  bytes.Buffer
	binaryPath string
	configPath string
	workingDir string
	workspace  string
	httpClient *http.Client
}

var (
	agentProcessManager *AgentProcessManager
	agentProcessOnce    sync.Once
)

func GetAgentProcessManager() *AgentProcessManager {
	agentProcessOnce.Do(func() {
		agentProcessManager = &AgentProcessManager{
			httpClient: &http.Client{Timeout: 1500 * time.Millisecond},
		}
	})
	return agentProcessManager
}

func (m *AgentProcessManager) Init(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ctx = ctx
}

func (m *AgentProcessManager) Status() AgentProcessStatus {
	m.mu.RLock()
	status := AgentProcessStatus{
		Managed:       m.cmd != nil,
		PID:           0,
		BinaryPath:    m.binaryPath,
		ConfigPath:    m.configPath,
		WorkingDir:    m.workingDir,
		WorkspacePath: m.workspace,
		APIBase:       GetAPIProxy().GetTargetBase(),
		LastError:     m.lastError,
		LastExit:      m.lastExit,
		OutputPreview: m.outputTailLocked(),
	}
	cmd := m.cmd
	startedAt := m.startedAt
	m.mu.RUnlock()

	if cmd != nil && cmd.Process != nil {
		status.PID = cmd.Process.Pid
		status.Running = processRunning(cmd.Process)
	}
	if !startedAt.IsZero() {
		status.StartedAt = startedAt.Format(time.RFC3339)
	}
	status.Healthy = m.checkHealth(status.APIBase)
	if status.Healthy && !status.Running {
		status.Running = true
	}

	return status
}

func (m *AgentProcessManager) Wake() (AgentProcessStatus, error) {
	if status := m.Status(); status.Healthy {
		return status, nil
	}

	m.mu.Lock()
	if m.cmd != nil && m.cmd.Process != nil && processRunning(m.cmd.Process) {
		status := m.statusLocked()
		m.mu.Unlock()
		return status, nil
	}

	binaryPath, configPath, workingDir, workspacePath, err := resolveAgentPaths()
	if err != nil {
		m.lastError = err.Error()
		status := m.statusLocked()
		m.mu.Unlock()
		return status, err
	}

	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		m.lastError = err.Error()
		status := m.statusLocked()
		m.mu.Unlock()
		return status, fmt.Errorf("failed to ensure workspace: %w", err)
	}

	args := []string{}
	if configPath != "" {
		args = append(args, "-c", configPath)
	}

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workingDir
	cmd.Env = append(os.Environ(), "ICOOCLAW_WORKSPACE="+workspacePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	m.outputBuf.Reset()
	cmd.Stdout = &m.outputBuf
	cmd.Stderr = &m.outputBuf

	if err := cmd.Start(); err != nil {
		m.lastError = err.Error()
		status := m.statusLocked()
		m.mu.Unlock()
		return status, fmt.Errorf("failed to start icoo_agent: %w", err)
	}

	m.cmd = cmd
	m.startedAt = time.Now()
	m.lastError = ""
	m.lastExit = ""
	m.binaryPath = binaryPath
	m.configPath = configPath
	m.workingDir = workingDir
	m.workspace = workspacePath
	m.mu.Unlock()

	go m.waitForExit(cmd)

	status, err := m.waitForHealthy(12 * time.Second)
	if err != nil {
		m.mu.Lock()
		m.lastError = err.Error()
		m.mu.Unlock()
		return status, err
	}
	return status, nil
}

func (m *AgentProcessManager) Stop() (AgentProcessStatus, error) {
	m.mu.Lock()
	cmd := m.cmd
	m.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		status := m.Status()
		if status.Healthy {
			return status, errors.New("icoo_agent 当前可达，但不是由 icoo_chat 托管，无法停止")
		}
		return status, errors.New("icoo_agent 未运行")
	}

	if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return m.Status(), fmt.Errorf("failed to stop icoo_agent: %w", err)
	}

	time.Sleep(300 * time.Millisecond)
	return m.Status(), nil
}

func (m *AgentProcessManager) Restart() (AgentProcessStatus, error) {
	_, stopErr := m.Stop()
	if stopErr != nil && !strings.Contains(stopErr.Error(), "未运行") {
		return m.Status(), stopErr
	}
	return m.Wake()
}

func (m *AgentProcessManager) Shutdown() {
	m.mu.Lock()
	cmd := m.cmd
	m.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return
	}

	if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		if m.ctx != nil {
			runtime.LogWarning(m.ctx, "failed to stop managed icoo_agent on shutdown: "+err.Error())
		}
	}
}

func (m *AgentProcessManager) waitForExit(cmd *exec.Cmd) {
	err := cmd.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd == cmd {
		m.cmd = nil
	}
	if err != nil {
		m.lastExit = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				m.lastExit = fmt.Sprintf("exit code %d", status.ExitStatus())
			}
		}
		if m.ctx != nil {
			runtime.LogWarning(m.ctx, "icoo_agent exited: "+m.lastExit)
		}
		return
	}
	m.lastExit = "exited normally"
}

func (m *AgentProcessManager) waitForHealthy(timeout time.Duration) (AgentProcessStatus, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status := m.Status()
		if status.Healthy {
			return status, nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return m.Status(), fmt.Errorf("icoo_agent started but health check did not pass within %s", timeout)
}

func (m *AgentProcessManager) checkHealth(apiBase string) bool {
	apiBase = strings.TrimRight(strings.TrimSpace(apiBase), "/")
	if apiBase == "" {
		return false
	}

	resp, err := m.httpClient.Get(apiBase + "/api/v1/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func (m *AgentProcessManager) statusLocked() AgentProcessStatus {
	status := AgentProcessStatus{
		Managed:       m.cmd != nil,
		BinaryPath:    m.binaryPath,
		ConfigPath:    m.configPath,
		WorkingDir:    m.workingDir,
		WorkspacePath: m.workspace,
		APIBase:       GetAPIProxy().GetTargetBase(),
		LastError:     m.lastError,
		LastExit:      m.lastExit,
		OutputPreview: m.outputTailLocked(),
	}
	if !m.startedAt.IsZero() {
		status.StartedAt = m.startedAt.Format(time.RFC3339)
	}
	if m.cmd != nil && m.cmd.Process != nil {
		status.PID = m.cmd.Process.Pid
		status.Running = processRunning(m.cmd.Process)
	}
	return status
}

func (m *AgentProcessManager) outputTailLocked() string {
	const maxLen = 1600
	text := strings.TrimSpace(m.outputBuf.String())
	if len(text) <= maxLen {
		return text
	}
	return text[len(text)-maxLen:]
}

func processRunning(process *os.Process) bool {
	return process != nil
}

func resolveAgentPaths() (binaryPath, configPath, workingDir, workspacePath string, err error) {
	configuredBinaryPath := strings.TrimSpace(GetConfigService().GetAgentProcessConfig().BinaryPath)
	if configuredBinaryPath != "" {
		configuredBinaryPath = filepath.Clean(configuredBinaryPath)
		if !fileExists(configuredBinaryPath) {
			return "", "", "", "", fmt.Errorf("配置的 icoo_agent 路径不存在: %s", configuredBinaryPath)
		}

		workingDir = filepath.Dir(filepath.Dir(configuredBinaryPath))
		if strings.EqualFold(filepath.Base(workingDir), "bin") {
			workingDir = filepath.Dir(workingDir)
		}

		configCandidate := filepath.Join(workingDir, "config.toml")
		if !fileExists(configCandidate) {
			configCandidate = ""
		}

		return configuredBinaryPath, configCandidate, workingDir, filepath.Join(workingDir, "workspace"), nil
	}

	cwd, _ := os.Getwd()
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	roots := uniquePaths([]string{
		cwd,
		filepath.Dir(cwd),
		exeDir,
		filepath.Dir(exeDir),
	})

	for _, root := range roots {
		candidates := []string{
			filepath.Join(root, "icoo_agent"),
			filepath.Join(root, "..", "icoo_agent"),
		}
		for _, candidate := range candidates {
			candidate = filepath.Clean(candidate)
			binCandidate := filepath.Join(candidate, "bin", "icooclaw.exe")
			if !fileExists(binCandidate) {
				continue
			}

			configCandidate := filepath.Join(candidate, "config.toml")
			if !fileExists(configCandidate) {
				configCandidate = ""
			}
			return binCandidate, configCandidate, candidate, filepath.Join(candidate, "workspace"), nil
		}
	}

	return "", "", "", "", fmt.Errorf("未找到 icoo_agent 可执行文件，期望路径类似 ../icoo_agent/bin/icooclaw.exe")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, item := range paths {
		item = filepath.Clean(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}
