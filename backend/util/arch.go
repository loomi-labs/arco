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

func IsDarwin() bool {
	return runtime.GOOS == Darwin.String()
}

func (o OS) String() string {
	return string(o)
}
