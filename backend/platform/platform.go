package platform

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/loomi-labs/arco/backend/util"
)

// GetGlibcVersion detects the system's GLIBC version on Linux systems
func GetGlibcVersion() (*version.Version, error) {
	if !util.IsLinux() {
		return nil, nil // Not applicable for non-Linux systems
	}

	cmd := exec.Command("ldd", "--version")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to detect GLIBC version: %w", err)
	}

	// Parse output like "ldd (GNU libc) 2.42" or "ldd (Ubuntu GLIBC 2.31-0ubuntu9.7) 2.31"
	// The version is typically the last word on the first line
	lines := strings.Split(string(output), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty ldd output")
	}

	firstLine := strings.TrimSpace(lines[0])
	fields := strings.Fields(firstLine)
	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields in ldd output")
	}

	// The version should be the last field and match the pattern x.y
	versionCandidate := fields[len(fields)-1]
	re := regexp.MustCompile(`^(\d+\.\d+)`)
	matches := re.FindStringSubmatch(versionCandidate)
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not parse GLIBC version from ldd output: %s", firstLine)
	}

	v, err := version.NewVersion(matches[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse GLIBC version %s: %w", matches[1], err)
	}

	return v, nil
}

// GetOpenFileManagerCmd returns the command to open the file manager for the current OS
func GetOpenFileManagerCmd() (string, error) {
	if util.IsLinux() {
		return "xdg-open", nil
	}
	if util.IsMacOS() {
		return "open", nil
	}
	return "", fmt.Errorf("operating system %s is not supported", runtime.GOOS)
}

// GetMountPath returns the default mount path for the current OS
func GetMountPath() (string, error) {
	if util.IsLinux() {
		return "/run/user", nil
	}
	if util.IsMacOS() {
		return "/private/tmp", nil
	}
	return "", fmt.Errorf("operating system %s is not supported", runtime.GOOS)
}