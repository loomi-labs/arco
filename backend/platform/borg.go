package platform

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/hashicorp/go-version"
)

// Binaries contains all available Borg binary variants
var Binaries = []BorgBinary{
	// Borg 1.4.1 - Linux variants
	{
		Name:         "borg_1.4.1",
		Version:      version.Must(version.NewVersion("1.4.1")),
		Os:           Linux,
		GlibcVersion: version.Must(version.NewVersion("2.28")),
		Url:          "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-linux-glibc228",
	},
	{
		Name:         "borg_1.4.1",
		Version:      version.Must(version.NewVersion("1.4.1")),
		Os:           Linux,
		GlibcVersion: version.Must(version.NewVersion("2.31")),
		Url:          "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-linux-glibc231",
	},
	{
		Name:         "borg_1.4.1",
		Version:      version.Must(version.NewVersion("1.4.1")),
		Os:           Linux,
		GlibcVersion: version.Must(version.NewVersion("2.36")),
		Url:          "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-linux-glibc236",
	},
	// Borg 1.4.1 - macOS
	{
		Name:         "borg_1.4.1",
		Version:      version.Must(version.NewVersion("1.4.1")),
		Os:           Darwin,
		GlibcVersion: nil,
		Url:          "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-macos1012",
	},
	// Borg 1.4.0 - Linux variants
	{
		Name:         "borg_1.4.0",
		Version:      version.Must(version.NewVersion("1.4.0")),
		Os:           Linux,
		GlibcVersion: version.Must(version.NewVersion("2.28")),
		Url:          "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-linux-glibc228",
	},
	{
		Name:         "borg_1.4.0",
		Version:      version.Must(version.NewVersion("1.4.0")),
		Os:           Linux,
		GlibcVersion: version.Must(version.NewVersion("2.31")),
		Url:          "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-linux-glibc231",
	},
	{
		Name:         "borg_1.4.0",
		Version:      version.Must(version.NewVersion("1.4.0")),
		Os:           Linux,
		GlibcVersion: version.Must(version.NewVersion("2.36")),
		Url:          "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-linux-glibc236",
	},
	// Borg 1.4.0 - macOS
	{
		Name:         "borg_1.4.0",
		Version:      version.Must(version.NewVersion("1.4.0")),
		Os:           Darwin,
		GlibcVersion: nil,
		Url:          "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-macos1012",
	},
}

// GetLatestBorgBinary selects the appropriate Borg binary for the current system
func GetLatestBorgBinary(binaries []BorgBinary) (BorgBinary, error) {
	// 1. Check if Linux or Darwin -> if not return error
	currentOS := OS(runtime.GOOS)
	if !IsLinux() && !IsMacOS() {
		return BorgBinary{}, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	// 2. Find latest binary for current OS
	var latestBinary BorgBinary
	var latestVersion *version.Version

	for _, binary := range binaries {
		if binary.Os == currentOS {
			if latestVersion == nil || binary.Version.GreaterThan(latestVersion) {
				latestBinary = binary
				latestVersion = binary.Version
			}
		}
	}

	if latestVersion == nil {
		return BorgBinary{}, fmt.Errorf("no binary found for operating system %s", runtime.GOOS)
	}

	// 3. If on Darwin, return this binary
	if IsMacOS() {
		return latestBinary, nil
	}

	// 4. Otherwise we are on Linux -> get glibc version
	systemGlibc, err := getGlibcVersion()
	if err != nil {
		// If GLIBC detection fails, fallback to lowest GLIBC requirement
		return selectLowestGlibcBinary(binaries), nil
	}

	// 5. Compare GLIBC versions -> select highest version that is <= system GLIBC version
	var bestBinary BorgBinary
	var bestGlibcVersion *version.Version

	for _, binary := range binaries {
		if binary.Os != currentOS || !binary.Version.Equal(latestVersion) {
			continue // Only consider binaries for current OS with latest Borg version
		}

		if binary.GlibcVersion != nil && binary.GlibcVersion.LessThanOrEqual(systemGlibc) {
			if bestGlibcVersion == nil || binary.GlibcVersion.GreaterThan(bestGlibcVersion) {
				bestBinary = binary
				bestGlibcVersion = binary.GlibcVersion
			}
		}
	}

	if bestGlibcVersion == nil {
		// No compatible GLIBC version found, return lowest available
		return selectLowestGlibcBinary(binaries), nil
	}

	return bestBinary, nil
}

// getGlibcVersion detects the system's GLIBC version on Linux systems
func getGlibcVersion() (*version.Version, error) {
	if !IsLinux() {
		return nil, fmt.Errorf("only Linux supports glibc") // Not applicable for non-Linux systems
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

// selectLowestGlibcBinary returns the binary with the lowest GLIBC requirement
func selectLowestGlibcBinary(binaries []BorgBinary) BorgBinary {
	if len(binaries) == 0 {
		return BorgBinary{}
	}

	lowest := binaries[0]

	for _, binary := range binaries[1:] {
		if binary.GlibcVersion == nil {
			continue
		}

		if lowest.GlibcVersion == nil || binary.GlibcVersion.LessThan(lowest.GlibcVersion) {
			lowest = binary
		}
	}

	return lowest
}
