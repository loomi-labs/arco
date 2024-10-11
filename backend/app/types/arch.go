package types

import (
	"fmt"
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

func getDarwinMountStates(paths map[int]string) (states map[int]*MountState, err error) {
	mountPath, err := GetMountPath()
	if err != nil {
		return
	}

	cmd := exec.Command("mount", "|", "grep", mountPath)
	output, err := cmd.Output()
	if err != nil {
		return
	}

	mountPoints := strings.Split(string(output), "\n")
	for _, mount := range mountPoints {
		for id, path := range paths {
			if mount == path {
				states[id] = &MountState{
					IsMounted: true,
					MountPath: mount,
				}
			}
		}
	}
	return
}

func getLinuxMountStates(paths map[int]string) (states map[int]*MountState, err error) {
	states = make(map[int]*MountState)

	mounts, err := procfs.GetMounts()
	if err != nil {
		return
	}

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
	return
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
