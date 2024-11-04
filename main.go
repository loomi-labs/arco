package main

import (
	"embed"
	"fmt"
	"github.com/loomi-labs/arco/backend/cmd"
	_ "github.com/loomi-labs/arco/backend/ent/runtime" // required to allow cyclic imports
	_ "github.com/mattn/go-sqlite3"
	"io"
	"io/fs"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed icon.svg
var icon embed.FS

//go:embed backend/ent/migrate/migrations
var migrations embed.FS

func main() {
	migrationsDir, err := fs.Sub(migrations, "backend/ent/migrate/migrations")
	if err != nil {
		panic(fmt.Errorf("failed to get migrations directory: %w", err))
	}

	iconFile, err := icon.Open("icon.svg")
	if err != nil {
		panic(fmt.Errorf("failed to open icon: %w", err))
	}
	iconData, err := io.ReadAll(iconFile)
	if err != nil {
		panic(fmt.Errorf("failed to read icon: %w", err))
	}
	err = iconFile.Close()
	if err != nil {
		panic(fmt.Errorf("failed to close icon: %w", err))
	}

	cmd.Execute(assets, iconData, migrationsDir)
}
