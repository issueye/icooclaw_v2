// Package builtin provides built-in tools for icooclaw.
package builtin

import (
	"os"

	"icooclaw/pkg/config"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/envmgr"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
	"icooclaw/pkg/tools/builtin/file"
	"icooclaw/pkg/tools/builtin/shell"
	"icooclaw/pkg/tools/builtin/web"
)

// RegisterBuiltinTools registers all built-in tools.
func RegisterBuiltinTools(registry *tools.Registry, st *storage.Storage, cfg *config.Config) {
	registry.Register(web.NewHTTPTool())
	registry.Register(web.NewWebSearchTool())
	registry.Register(NewDateTimeTool())
	registry.Register(NewListAgentsTool(st))
	registry.Register(NewGetAgentTool(st))
	registry.Register(NewCreateAgentTool(st))
	registry.Register(NewUpdateAgentTool(st))
	registry.Register(NewDeleteAgentTool(st))
	registry.Register(NewSearchMessagesTool(st))

	// 文件系统工具
	// 使用环境变量或默认工作目录
	workDir := os.Getenv("ICOOCALW_WORKSPACE")
	if workDir == "" {
		workDir = "./workspace"
	}

	// 注册综合文件系统工具
	registry.Register(file.NewFilesystemTool(workDir))

	// 注册独立的文件操作工具
	registry.Register(file.NewReadFileTool(workDir))
	registry.Register(file.NewWriteFileTool(workDir))
	registry.Register(file.NewListDirTool(workDir))
	registry.Register(file.NewCopyFileTool(workDir))
	registry.Register(file.NewSearchCodeTool(workDir))
	registry.Register(file.NewReplaceInFileTool(workDir))
	registry.Register(file.NewInsertInFileTool(workDir))

	// 注册 shell 命令工具
	registry.Register(shell.NewShellCommandTool(
		shell.WithWorkDir(workDir),
		shell.WithTimeout(resolveExecTimeout(cfg)),
		shell.WithDefaultEnv(resolveExecEnv(cfg)),
		shell.WithEnvProvider(resolveRuntimeExecEnv(st)),
	))
}

func resolveExecTimeout(cfg *config.Config) int {
	if cfg == nil || cfg.Exec.Timeout <= 0 {
		return 60
	}
	return cfg.Exec.Timeout
}

func resolveExecEnv(cfg *config.Config) map[string]string {
	if cfg == nil {
		return map[string]string{}
	}
	return cfg.Exec.Env
}

func resolveRuntimeExecEnv(st *storage.Storage) func() map[string]string {
	return func() map[string]string {
		if st == nil {
			return map[string]string{}
		}

		if st.ExecEnv() != nil {
			values, err := st.ExecEnv().ToMap()
			if err == nil && len(values) > 0 {
				return values
			}
		}

		if st.Param() == nil {
			return map[string]string{}
		}
		param, err := st.Param().Get(consts.EXEC_ENV_KEY)
		if err != nil || param == nil || !param.Enabled {
			return map[string]string{}
		}
		values, err := envmgr.ParseJSON(param.Value)
		if err != nil {
			return map[string]string{}
		}
		return values
	}
}
