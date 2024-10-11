package types

import (
	"fmt"
	"runtime"
)

type OS string

const (
	Linux  OS = "linux"
	Darwin OS = "darwin"
)

func (o OS) String() string {
	return string(o)
}

type Binary struct {
	Name    string
	Version string
	Os      OS
	Url     string
}

func IsLinux() bool {
	return runtime.GOOS == Linux.String()
}

func IsDarwin() bool {
	return runtime.GOOS == Darwin.String()
}

func GetLatestBorgBinary(binaries []Binary) (Binary, error) {
	for _, binary := range binaries {
		if binary.Os == OS(runtime.GOOS) {
			return binary, nil
		}
	}
	return Binary{}, fmt.Errorf("no binary found for operating system %s", runtime.GOOS)
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

func GetMountPath() (string, error) {
	if IsLinux() {
		return "/proc/self/mountinfo", nil
	}
	if IsDarwin() {
		return "/Volumes", nil
	}
	return "", fmt.Errorf("operating system %s is not supported", runtime.GOOS)
}
