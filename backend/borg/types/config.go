package types

import "embed"

type Config struct {
	Binaries    embed.FS
	BorgPath    string
	BorgVersion string
	Icon        embed.FS
}
