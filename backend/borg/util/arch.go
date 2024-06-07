package util

import "runtime"

func GetBinaryPath() string {
	if runtime.GOOS == "linux" {
		return "bin/borg-linuxnewer64"
	}
	if runtime.GOOS == "darwin" {
		return "bin/borg-macos64"
	}
	panic("unsupported OS")
}
