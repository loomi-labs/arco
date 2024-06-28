package types

import "embed"

type Config struct {
	Dir         string
	Binaries    embed.FS
	BorgPath    string
	BorgVersion string
	Icon        embed.FS
}
