package app

import (
	"os"

	"icooclaw/pkg/config"
	"icooclaw/pkg/logging"
	"log/slog"
)

// InitConfig 初始化配置
func (a *App) InitConfig(cfgFile string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		return err
	}

	if err := cfg.EnsureWorkspaceScaffold(); err != nil {
		slog.Error("创建工作区骨架失败", "error", err)
		return err
	}
	if err := cfg.EnsureDatabasePath(); err != nil {
		slog.Error("创建数据库目录失败", "error", err)
		return err
	}

	a.Cfg = cfg
	return nil
}

// InitLog 初始化日志记录器
func (a *App) InitLog() *slog.Logger {
	logger, closer, err := logging.NewLogger(logging.Options{
		Level:     parseLogLevel(a.Cfg.Logging.Level),
		Format:    a.Cfg.Logging.Format,
		AddSource: a.Cfg.Logging.AddSource,
		Output:    a.Cfg.Logging.Path,
		Stdout:    os.Stdout,
	})
	if err != nil {
		slog.Error("初始化日志失败，回退到标准输出", "error", err)
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: parseLogLevel(a.Cfg.Logging.Level),
		}))
	} else {
		a.LogCloser = closer
	}

	slog.SetDefault(logger)
	return logger
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
