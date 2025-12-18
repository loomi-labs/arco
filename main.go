package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"

	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/cmd"
	_ "github.com/loomi-labs/arco/backend/ent/runtime" // required to allow cyclic imports
	_ "github.com/mattn/go-sqlite3"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon-dark.png
var appIconDarkFs embed.FS

//go:embed build/appicon-light.png
var appIconLightFs embed.FS

//go:embed build/darwin/icons.icns
var darwinIconsFs embed.FS

//go:embed build/darwin/menubar-icon.png
var darwinMenubarIconFs embed.FS

//go:embed build/windows/icon.ico
//var windowsIconFs embed.FS

//go:embed backend/ent/migrate/migrations
var migrations embed.FS

func readEmbeddedFile(embeddedFS embed.FS, path string) []byte {
	file, err := embeddedFS.Open(path)
	if err != nil {
		panic(fmt.Errorf("failed to open file %s: %w", path, err))
	}
	defer func(file fs.File) {
		err := file.Close()
		if err != nil {
			panic(fmt.Errorf("failed to close file %s: %w", path, err))
		}
	}(file)

	content, err := io.ReadAll(file)
	if err != nil {
		panic(fmt.Errorf("failed to read file %s: %w", path, err))
	}

	return content
}

func getIcons() *types.Icons {
	appIconDark := readEmbeddedFile(appIconDarkFs, "build/appicon-dark.png")
	appIconLight := readEmbeddedFile(appIconLightFs, "build/appicon-light.png")
	darwinIcons := readEmbeddedFile(darwinIconsFs, "build/darwin/icons.icns")
	darwinMenubarIcon := readEmbeddedFile(darwinMenubarIconFs, "build/darwin/menubar-icon.png")

	return &types.Icons{
		AppIconDark:       appIconDark,
		AppIconLight:      appIconLight,
		DarwinIcons:       darwinIcons,
		DarwinMenubarIcon: darwinMenubarIcon,
	}
}

func getMigrations() fs.FS {
	migrationsDir, err := fs.Sub(migrations, "backend/ent/migrate/migrations")
	if err != nil {
		panic(fmt.Errorf("failed to get migrations directory: %w", err))
	}
	return migrationsDir
}

func main() {
	cmd.Execute(assets, getIcons(), getMigrations())
}
