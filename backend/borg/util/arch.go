package util

import (
	"fmt"
	"runtime"
)

func GetBinaryPathX() string {
	if runtime.GOOS == "linux" {
		return "bin/borg-linuxnewer64"
	}
	if runtime.GOOS == "darwin" {
		return "bin/borg-macos64"
	}
	panic("unsupported OS")
}

func GetOpenFileManagerCmd() (string, error) {
	if runtime.GOOS == "linux" {
		return "xdg-open", nil
	}
	if runtime.GOOS == "darwin" {
		return "open", nil
	}
	return "", fmt.Errorf("operating system %s is not supported", runtime.GOOS)
}
