package platform

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/prometheus/procfs"
)

// GetMountPath returns the default mount path for the current OS
func GetMountPath() (string, error) {
	if IsLinux() {
		return "/run/user", nil
	}
	if IsMacOS() {
		return "/private/tmp", nil
	}
	return "", fmt.Errorf("operating system %s is not supported", runtime.GOOS)
}

func GetMountStates(paths map[int]string) (states map[int]*MountState, err error) {
	if IsLinux() {
		return getLinuxMountStates(paths)
	}
	if IsMacOS() {
		return getDarwinMountStates(paths)
	}
	return nil, fmt.Errorf("operating system %s is not supported", runtime.GOOS)
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
