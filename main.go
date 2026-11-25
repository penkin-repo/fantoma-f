package main

import (
	"context"
	"embed"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

// trayIconData embeds the real application icon for the system tray.
//
//go:embed build/windows/icon.ico
var trayIconData []byte

var globalApp *App

// exitOnce ensures handleExit runs only once, even if called from multiple goroutines.
var exitOnce sync.Once

func main() {
	startMinimized := false
	for _, arg := range os.Args[1:] {
		if arg == "--minimized" {
			startMinimized = true
		}
	}

	// Read saved theme before launching window so initial frame & titlebar match without flash.
	initialTheme := "dark"
	if cfg, err := loadConfig(); err == nil {
		initialTheme = cfg.effectiveTheme()
	}

	winTheme := windows.Dark
	bgColour := &options.RGBA{R: 17, G: 18, B: 20, A: 255}
	if initialTheme == "light" {
		winTheme = windows.Light
		bgColour = &options.RGBA{R: 253, G: 252, B: 249, A: 255}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		handleExit()
	}()

	globalApp = NewApp()

	err := wails.Run(&options.App{
		Title:             "Fantoma F " + AppVersion,
		Width:             1280,
		Height:            850,
		MinWidth:          900,
		MinHeight:         650,
		StartHidden:       startMinimized,
		HideWindowOnClose: true,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "fantoma-f-font-manager-v1",
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				if globalApp != nil && globalApp.ctx != nil {
					runtime.WindowUnminimise(globalApp.ctx)
					runtime.WindowShow(globalApp.ctx)
				}
			},
		},
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: fontHandler(),
		},
		BackgroundColour: bgColour,
		OnStartup: func(ctx context.Context) {
			globalApp.startup(ctx)
			actionCh := StartNativeTray(trayIconData)
			go handleTrayActions(actionCh)
		},
		OnBeforeClose: func(ctx context.Context) bool {
			if globalApp != nil {
				go globalApp.TrimMemory()
			}
			return false
		},
		OnShutdown: globalApp.shutdown,
		Bind: []interface{}{
			globalApp,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			Theme:                winTheme,
		},
	})
	if err != nil {
		log.Fatal("Wails error:", err)
	}
}

func handleTrayActions(ch chan TrayAction) {
	for action := range ch {
		switch action {
		case TrayShow:
			go func() {
				if globalApp != nil && globalApp.ctx != nil {
					runtime.WindowUnminimise(globalApp.ctx)
					runtime.WindowShow(globalApp.ctx)
				}
			}()
		case TrayDeactivateAll:
			go func() {
				if globalApp != nil {
					_ = globalApp.DeactivateAll()
				}
			}()
		case TrayQuit:
			handleExit()
		}
	}
}

// handleExit removes staged fonts from Windows (preserving config.json ActiveFonts
// so they restore on next launch) and exits cleanly.
func handleExit() {
	exitOnce.Do(func() {
		done := make(chan struct{})
		go func() {
			if globalApp != nil {
				globalApp.deactivateSystemFontsOnly()
			}
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}

		StopNativeTray()

		if globalApp != nil && globalApp.ctx != nil {
			runtime.Quit(globalApp.ctx)
		}
		os.Exit(0)
	})
}

// fontHandler serves local TTF/OTF font files for WebView2 preview.
func fontHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/font" {
			fontPath := r.URL.Query().Get("path")
			if fontPath == "" {
				http.Error(w, "missing path parameter", http.StatusBadRequest)
				return
			}

			ext := strings.ToLower(filepath.Ext(fontPath))
			if ext != ".ttf" && ext != ".otf" {
				http.Error(w, "unsupported file type", http.StatusForbidden)
				return
			}

			stat, err := os.Stat(fontPath)
			if err != nil || stat.IsDir() {
				http.Error(w, "file not found or is a directory", http.StatusNotFound)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Cache-Control", "public, max-age=31536000")

			http.ServeFile(w, r, fontPath)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
}
