package platform

import (
	"fmt"
	"runtime"
)

func IsLinux() bool {
	return runtime.GOOS == Linux.String()
}

func IsMacOS() bool {
	return runtime.GOOS == Darwin.String()
}

func GithubAssetName() string {
	if IsLinux() {
		return "arco-linux.zip"
	}
	if IsMacOS() {
		return "arco-macos.zip"
	}
	return ""
}

// GetOpenFileManagerCmd returns the command to open the file manager for the current OS
func GetOpenFileManagerCmd() (string, error) {
	if IsLinux() {
		return "xdg-open", nil
	}
	if IsMacOS() {
		return "open", nil
	}
	return "", fmt.Errorf("operating system %s is not supported", runtime.GOOS)
}
