package types

import (
	"fmt"
	"github.com/loomi-labs/arco/backend/util"
	"github.com/prometheus/procfs"
	"os/exec"
	"runtime"
	"strings"
)

type Binary struct {
	Name    string
	Version string
	Os      util.OS
	Url     string
}

func GetLatestBorgBinary(binaries []Binary) (Binary, error) {
	for _, binary := range binaries {
		if binary.Os == util.OS(runtime.GOOS) {
			return binary, nil
		}
	}
	return Binary{}, fmt.Errorf("no binary found for operating system %s", runtime.GOOS)
}

func GetOpenFileManagerCmd() (string, error) {
	if util.IsLinux() {
		return "xdg-open", nil
	}
	if util.IsDarwin() {
		return "open", nil
	}
	return "", fmt.Errorf("operating system %s is not supported", runtime.GOOS)
}

func GetMountPath() (string, error) {
	if util.IsLinux() {
		return "/run/user", nil
	}
	if util.IsDarwin() {
		return "/private/tmp", nil
	}
	return "", fmt.Errorf("operating system %s is not supported", runtime.GOOS)
}

func getDarwinMountStates(paths map[int]string) (map[int]*MountState, error) {

	cmd := exec.Command("mount")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error running mount command: %s", err)
	}

	states := make(map[int]*MountState)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		for id, path := range paths {
			parts := strings.Fields(line)
			if len(parts) > 2 && parts[2] == path {
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
	if util.IsLinux() {
		return getLinuxMountStates(paths)
	}
	if util.IsDarwin() {
		return getDarwinMountStates(paths)
	}
	return nil, fmt.Errorf("operating system %s is not supported", runtime.GOOS)
}
