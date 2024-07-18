package util

import (
	"arco/backend/types"
	"fmt"
	"runtime"
)

func IsLinux() bool {
	return runtime.GOOS == types.Linux.String()
}

func IsDarwin() bool {
	return runtime.GOOS == types.Darwin.String()
}

func GetLatestBorgBinary(binaries []types.Binary) (types.Binary, error) {
	for _, binary := range binaries {
		if binary.Os == types.OS(runtime.GOOS) {
			return binary, nil
		}
	}
	return types.Binary{}, fmt.Errorf("no binary found for operating system %s", runtime.GOOS)
}

func GetOpenFileManagerCmd() (string, error) {
	if IsLinux() {
		return "xdg-open", nil
	}
	if IsDarwin() {
		return "open", nil
	}
	return "", fmt.Errorf("operating system %s is not supported", runtime.GOOS)
}
