package types

import "embed"

type Config struct {
	Dir         string
	Binaries    []Binary
	BorgPath    string
	BorgVersion string
	Icon        embed.FS
}

type Binary struct {
	Name    string
	Version string
	Os      OS
	Url     string
}
