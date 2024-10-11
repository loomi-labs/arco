package types

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/prometheus/procfs"
	"os/exec"
	"runtime"
	"strings"
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
		return "/run/user", nil
	}
	if IsDarwin() {
		return "/private/tmp", nil
	}
	return "", fmt.Errorf("operating system %s is not supported", runtime.GOOS)
}

func getDarwinMountStates(paths map[int]string) (map[int]*MountState, error) {

	cmd := exec.Command("mount")
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Errorf("error running mount command: %s", err)
	}

	states := make(map[int]*MountState)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		for id, path := range paths {
			if strings.Contains(line, path) {
				states[id] = &MountState{
					IsMounted: true,
					MountPath: path,
				}
			}
		}
	}
	return states, nil
}

func getLinuxMountStates(paths map[int]string) (map[int]*MountState, error) {
	mounts, err := procfs.GetMounts()
	if err != nil {
		return nil, err
	}

	states := make(map[int]*MountState)
	for _, mount := range mounts {
		for id, path := range paths {
			if mount.MountPoint == path {
				states[id] = &MountState{
					IsMounted: true,
					MountPath: mount.MountPoint,
				}
			}
		}
	}
	return states, nil
}

func GetMountStates(paths map[int]string) (states map[int]*MountState, err error) {
	if IsLinux() {
		return getLinuxMountStates(paths)
	}
	if IsDarwin() {
		return getDarwinMountStates(paths)
	}
	return nil, fmt.Errorf("operating system %s is not supported", runtime.GOOS)
}
