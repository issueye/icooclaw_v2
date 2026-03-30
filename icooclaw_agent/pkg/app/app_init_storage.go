package app

import (
	"os"

	"icooclaw/pkg/storage"
	"log/slog"
)

// InitStorage 初始化存储
func (a *App) InitStorage() {
	dbPath, _ := a.Cfg.GetDatabasePath()
	store, err := storage.NewWithOptions(a.Cfg.Workspace, a.Cfg.Mode, dbPath, storage.MessageStoreOptions{
		Type: a.Cfg.MessageStore.Type,
		Path: a.Cfg.MessageStore.Path,
		Dir:  a.Cfg.MessageStore.Dir,
	})
	if err != nil {
		slog.Error("初始化存储失败", "error", err)
		os.Exit(1)
	}

	a.Storage = store
}
