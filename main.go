package main

import (
	"arco/backend/cmd"
	_ "arco/backend/ent/runtime" // required to allow cyclic imports
	"embed"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io/fs"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed icon.png
var icon embed.FS

//go:embed backend/ent/migrate/migrations
var migrations embed.FS

func main() {
	migrationsDir, err := fs.Sub(migrations, "backend/ent/migrate/migrations")
	if err != nil {
		panic(fmt.Errorf("failed to get migrations directory: %w", err))
	}
	cmd.Execute(assets, icon, migrationsDir)
}
