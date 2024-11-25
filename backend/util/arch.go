package util

import "runtime"

type OS string

const (
	Linux  OS = "linux"
	Darwin OS = "darwin"
)

func IsLinux() bool {
	return runtime.GOOS == Linux.String()
}

func IsMacOS() bool {
	return runtime.GOOS == Darwin.String()
}

func (o OS) String() string {
	return string(o)
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
