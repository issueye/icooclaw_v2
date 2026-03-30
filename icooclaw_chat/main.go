package main

import (
	"embed"
	"icoo_chat/internal/services"
	"io/fs"
	"net/http"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

type assetHandler struct {
	assets http.Handler
	proxy  *services.APIProxy
}

func newAssetHandler() *assetHandler {
	sub, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		panic(err)
	}
	return &assetHandler{
		assets: http.FileServer(http.FS(sub)),
		proxy:  services.GetAPIProxy(),
	}
}

func (h *assetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/proxy/") {
		h.proxy.ServeHTTP(w, r)
		return
	}
	h.assets.ServeHTTP(w, r)
}

func main() {
	app := services.NewApp()

	err := wails.Run(&options.App{
		Title:  "icoo_chat",
		Width:  1100,
		Height: 890,
		AssetServer: &assetserver.Options{
			Handler: newAssetHandler(),
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		OnShutdown:       app.Shutdown,
		Bind: []interface{}{
			app,
		},
		Frameless: true,
		Windows: &windows.Options{
			DisableFramelessWindowDecorations: true,
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               true,
		},
	})

	if err != nil {
		println("错误:", err.Error())
	}
}
