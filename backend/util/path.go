package util

import (
	"os"
	"strings"
)

// ExpandPath expands a path that starts with ~ to the user's home directory
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		path = home + path[1:]
	}
	return path
}
