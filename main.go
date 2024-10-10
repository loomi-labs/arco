package main

import (
	"arco/backend/cmd"
	_ "arco/backend/ent/runtime" // required to allow cyclic imports
	"embed"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed icon.png
var icon embed.FS

func main() {
	cmd.Execute(assets, icon)
}
