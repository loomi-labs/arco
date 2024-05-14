package main

import (
	"embed"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"os"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logLevel := logger.INFO
	if os.Getenv("DEBUG") == "true" {
		logLevel = logger.DEBUG
	}

	log := logger.NewDefaultLogger()

	// Create an instance of the app structure
	app := NewApp(log)

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "timebender",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app.Borg,
		},
		LogLevel: logLevel,
		Logger:   log,
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
