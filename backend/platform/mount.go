package platform

import (
	"fmt"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
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

// GetArcoMounts returns all Arco repository and archive mounts in a single system call
// This is much more efficient than checking individual paths when dealing with many archives
func GetArcoMounts() (repos map[int]*MountState, archives map[int]*MountState, err error) {
	if IsLinux() {
		return getLinuxArcoMounts()
	}
	if IsMacOS() {
		return getDarwinArcoMounts()
	}
	return nil, nil, fmt.Errorf("operating system %s is not supported", runtime.GOOS)
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

// getDarwinArcoMounts efficiently finds all Arco mounts on macOS
func getDarwinArcoMounts() (repos map[int]*MountState, archives map[int]*MountState, err error) {
	// Get current user for path matching
	currentUser, err := user.Current()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get current user: %w", err)
	}

	// Get mount base path
	mountPath, err := GetMountPath()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get mount path: %w", err)
	}

	// Build expected path prefix for Arco mounts
	arcoPrefix := filepath.Join(mountPath, currentUser.Uid, "arco") + "/"

	// Compile regex patterns for repo and archive ID extraction
	repoPattern := regexp.MustCompile(`^` + regexp.QuoteMeta(arcoPrefix) + `repo-(\d+)$`)
	archivePattern := regexp.MustCompile(`^` + regexp.QuoteMeta(arcoPrefix) + `archive-(\d+)$`)

	// Get all mount points
	cmd := exec.Command("mount")
	output, err := cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("error running mount command: %w", err)
	}

	repos = make(map[int]*MountState)
	archives = make(map[int]*MountState)

	// Parse mount output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) <= 2 {
			continue
		}
		mountPoint := parts[2]

		// Check for repository mount
		if matches := repoPattern.FindStringSubmatch(mountPoint); matches != nil {
			if repoID, parseErr := strconv.Atoi(matches[1]); parseErr == nil {
				repos[repoID] = &MountState{
					IsMounted: true,
					MountPath: mountPoint,
				}
			}
		}

		// Check for archive mount
		if matches := archivePattern.FindStringSubmatch(mountPoint); matches != nil {
			if archiveID, parseErr := strconv.Atoi(matches[1]); parseErr == nil {
				archives[archiveID] = &MountState{
					IsMounted: true,
					MountPath: mountPoint,
				}
			}
		}
	}

	return repos, archives, nil
}

// getLinuxArcoMounts efficiently finds all Arco mounts on Linux
func getLinuxArcoMounts() (repos map[int]*MountState, archives map[int]*MountState, err error) {
	// Get current user for path matching
	currentUser, err := user.Current()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get current user: %w", err)
	}

	// Get mount base path
	mountPath, err := GetMountPath()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get mount path: %w", err)
	}

	// Build expected path prefix for Arco mounts
	arcoPrefix := filepath.Join(mountPath, currentUser.Uid, "arco") + "/"

	// Compile regex patterns for repo and archive ID extraction
	repoPattern := regexp.MustCompile(`^` + regexp.QuoteMeta(arcoPrefix) + `repo-(\d+)$`)
	archivePattern := regexp.MustCompile(`^` + regexp.QuoteMeta(arcoPrefix) + `archive-(\d+)$`)

	// Get all mount points using procfs
	mounts, err := procfs.GetMounts()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get mounts: %w", err)
	}

	repos = make(map[int]*MountState)
	archives = make(map[int]*MountState)

	// Process each mount point
	for _, mount := range mounts {
		mountPoint := mount.MountPoint

		// Check for repository mount
		if matches := repoPattern.FindStringSubmatch(mountPoint); matches != nil {
			if repoID, parseErr := strconv.Atoi(matches[1]); parseErr == nil {
				repos[repoID] = &MountState{
					IsMounted: true,
					MountPath: mountPoint,
				}
			}
		}

		// Check for archive mount
		if matches := archivePattern.FindStringSubmatch(mountPoint); matches != nil {
			if archiveID, parseErr := strconv.Atoi(matches[1]); parseErr == nil {
				archives[archiveID] = &MountState{
					IsMounted: true,
					MountPath: mountPoint,
				}
			}
		}
	}

	return repos, archives, nil
}
