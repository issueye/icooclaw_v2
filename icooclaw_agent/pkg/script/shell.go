// Package script provides JavaScript scripting engine for icooclaw.
package script

import (
	"bytes"
	"context"
	"fmt"
	"icooclaw/pkg/envmgr"
	"icooclaw/pkg/utils"
	"log/slog"
	"os"
	"os/exec"
	"time"
)

// ShellExec provides shell command execution.
type ShellExec struct {
	ctx    context.Context
	cfg    *Config
	logger *slog.Logger
}

// NewShellExec creates a new ShellExec builtin.
func NewShellExec(ctx context.Context, cfg *Config, logger *slog.Logger) *ShellExec {
	if logger == nil {
		logger = slog.Default()
	}
	return &ShellExec{ctx: ctx, cfg: cfg, logger: logger}
}

// Name returns the builtin name.
func (s *ShellExec) Name() string {
	return "shell"
}

// Object returns the shell object.
func (s *ShellExec) Object() map[string]any {
	return map[string]any{
		"exec":            s.Exec,
		"execWithTimeout": s.ExecWithTimeout,
		"execInDir":       s.ExecInDir,
	}
}

// Exec executes a command.
func (s *ShellExec) Exec(command string) (map[string]any, error) {
	return s.ExecWithTimeout(command, s.cfg.ExecTimeout)
}

// ExecWithTimeout executes a command with timeout.
func (s *ShellExec) ExecWithTimeout(command string, timeoutSeconds int) (map[string]any, error) {
	if !s.cfg.AllowExec {
		return nil, fmt.Errorf(errShellNotAllowed)
	}

	if timeoutSeconds <= 0 {
		timeoutSeconds = s.cfg.ExecTimeout
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultExecTimeoutSeconds
	}

	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	shellSpec := utils.DetectShell(command)
	cmd := exec.CommandContext(ctx, shellSpec.Command, shellSpec.Args...)
	cmd.Dir = s.cfg.Workspace
	cmd.Env = append(os.Environ(), envmgr.New(s.cfg.ExecEnv).ToList()...)
	cmd.Env = utils.MergeEnvWithWorkspaceBin(cmd.Env, cmd.Dir)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime).String()

	result := map[string]any{
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
		"duration":  duration,
		"success":   err == nil,
		"timed_out": ctx.Err() == context.DeadlineExceeded,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result["exit_code"] = exitErr.ExitCode()
		} else {
			result["exit_code"] = -1
			result["error"] = err.Error()
		}
	} else {
		result["exit_code"] = 0
	}

	return result, nil
}

// ExecInDir executes a command in a specific directory.
func (s *ShellExec) ExecInDir(command, workDir string) (map[string]any, error) {
	if !s.cfg.AllowExec {
		return nil, fmt.Errorf(errShellNotAllowed)
	}

	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := time.Duration(s.cfg.ExecTimeout) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	shellSpec := utils.DetectShell(command)
	cmd := exec.CommandContext(ctx, shellSpec.Command, shellSpec.Args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), envmgr.New(s.cfg.ExecEnv).ToList()...)
	cmd.Env = utils.MergeEnvWithWorkspaceBin(cmd.Env, cmd.Dir)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime).String()

	result := map[string]any{
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
		"duration":  duration,
		"success":   err == nil,
		"timed_out": ctx.Err() == context.DeadlineExceeded,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result["exit_code"] = exitErr.ExitCode()
		} else {
			result["exit_code"] = -1
			result["error"] = err.Error()
		}
	} else {
		result["exit_code"] = 0
	}

	return result, nil
}
