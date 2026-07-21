package platform

import (
	"github.com/hashicorp/go-version"
)

type OS string

const (
	Linux  OS = "linux"
	Darwin OS = "darwin"
)

func (o OS) String() string {
	return string(o)
}

// BorgBinary represents a Borg backup binary for a specific OS and platform requirement (GLIBC / macOS version)
type BorgBinary struct {
	Name          string
	Version       *version.Version
	Os            OS
	GlibcVersion  *version.Version // Minimum GLIBC version required. Only applicable for Linux, nil for non-Linux
	MacOSVersion  *version.Version // Minimum macOS version required. Only applicable for Darwin, nil means compatible with any macOS
	Arch          string           // CPU architecture (amd64, arm64), empty means any
	Url           string
	IsDirectory   bool // True for .tgz directory distributions (faster on macOS)
	SupportsMount bool // True for binaries with FUSE support (single binaries, not -gh directory builds)
}

// MountState represents the mount status of a repository
type MountState struct {
	IsMounted bool   `json:"isMounted"`
	MountPath string `json:"mountPath"`
}
