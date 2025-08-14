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

// BorgBinary represents a Borg backup binary for a specific OS and GLIBC version
type BorgBinary struct {
	Name         string
	Version      *version.Version
	Os           OS
	GlibcVersion *version.Version // Only applicable for Linux, nil for non-Linux
	Url          string
}

// MountState represents the mount status of a repository
type MountState struct {
	IsMounted bool   `json:"isMounted"`
	MountPath string `json:"mountPath"`
}
