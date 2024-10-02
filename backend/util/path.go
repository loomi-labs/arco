package util

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath expands a path that starts with ~ to the user's home directory
func ExpandPath(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	home = strings.TrimRight(home, string(os.PathSeparator))
	return filepath.Join(home, path[1:])
}
