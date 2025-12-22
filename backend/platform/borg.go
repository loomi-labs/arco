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
	// Borg 1.4.3 - Linux x86_64 variants
	{
		Name:          "borg_1.4.3",
		Version:       version.Must(version.NewVersion("1.4.3")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.31")),
		Arch:          "amd64",
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.3/borg-linux-glibc231-x86_64",
		SupportsMount: true,
	},
	{
		Name:          "borg_1.4.3",
		Version:       version.Must(version.NewVersion("1.4.3")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.35")),
		Arch:          "amd64",
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.3/borg-linux-glibc235-x86_64-gh",
		SupportsMount: true,
	},
	// Borg 1.4.3 - Linux ARM64
	{
		Name:          "borg_1.4.3",
		Version:       version.Must(version.NewVersion("1.4.3")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.35")),
		Arch:          "arm64",
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.3/borg-linux-glibc235-arm64-gh",
		SupportsMount: true,
	},
	// Borg 1.4.3 - macOS Intel (directory distribution for faster startup, no FUSE support)
	{
		Name:          "borg_1.4.3",
		Version:       version.Must(version.NewVersion("1.4.3")),
		Os:            Darwin,
		Arch:          "amd64",
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.3/borg-macos-13-x86_64-gh.tgz",
		IsDirectory:   true,
		SupportsMount: false, // -gh builds don't include llfuse
	},
	// Borg 1.4.3 - macOS Apple Silicon (directory distribution for faster startup, no FUSE support)
	{
		Name:          "borg_1.4.3",
		Version:       version.Must(version.NewVersion("1.4.3")),
		Os:            Darwin,
		Arch:          "arm64",
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.3/borg-macos-14-arm64-gh.tgz",
		IsDirectory:   true,
		SupportsMount: false, // -gh builds don't include llfuse
	},
	// Borg 1.4.1 - Linux variants
	{
		Name:          "borg_1.4.1",
		Version:       version.Must(version.NewVersion("1.4.1")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.28")),
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-linux-glibc228",
		SupportsMount: true,
	},
	{
		Name:          "borg_1.4.1",
		Version:       version.Must(version.NewVersion("1.4.1")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.31")),
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-linux-glibc231",
		SupportsMount: true,
	},
	{
		Name:          "borg_1.4.1",
		Version:       version.Must(version.NewVersion("1.4.1")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.36")),
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-linux-glibc236",
		SupportsMount: true,
	},
	// Borg 1.4.1 - macOS (single binary with FUSE support)
	{
		Name:          "borg_1.4.1",
		Version:       version.Must(version.NewVersion("1.4.1")),
		Os:            Darwin,
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-macos1012",
		SupportsMount: true, // Single binary includes llfuse
	},
	// Borg 1.4.0 - Linux variants
	{
		Name:          "borg_1.4.0",
		Version:       version.Must(version.NewVersion("1.4.0")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.28")),
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-linux-glibc228",
		SupportsMount: true,
	},
	{
		Name:          "borg_1.4.0",
		Version:       version.Must(version.NewVersion("1.4.0")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.31")),
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-linux-glibc231",
		SupportsMount: true,
	},
	{
		Name:          "borg_1.4.0",
		Version:       version.Must(version.NewVersion("1.4.0")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.36")),
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-linux-glibc236",
		SupportsMount: true,
	},
	// Borg 1.4.0 - macOS (single binary with FUSE support)
	{
		Name:          "borg_1.4.0",
		Version:       version.Must(version.NewVersion("1.4.0")),
		Os:            Darwin,
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-macos1012",
		SupportsMount: true, // Single binary includes llfuse
	},
}

// GetLatestBorgBinary selects the appropriate Borg binary for the current system
func GetLatestBorgBinary(binaries []BorgBinary) (BorgBinary, error) {
	// 1. Check if Linux or Darwin -> if not return error
	currentOS := OS(runtime.GOOS)
	currentArch := runtime.GOARCH
	if !IsLinux() && !IsMacOS() {
		return BorgBinary{}, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	// 2. Find latest binary for current OS and architecture
	var latestBinary BorgBinary
	var latestVersion *version.Version

	for _, binary := range binaries {
		// Check OS compatibility
		if binary.Os != currentOS {
			continue
		}

		// Check architecture compatibility
		// Empty Arch means compatible with any architecture (backward compatibility)
		if binary.Arch != "" && binary.Arch != currentArch {
			continue
		}

		// Track the latest version
		if latestVersion == nil || binary.Version.GreaterThan(latestVersion) {
			latestBinary = binary
			latestVersion = binary.Version
		}
	}

	if latestVersion == nil {
		return BorgBinary{}, fmt.Errorf("no binary found for operating system %s and architecture %s", runtime.GOOS, currentArch)
	}

	// 3. If on Darwin, return the architecture-matched binary
	if IsMacOS() {
		return latestBinary, nil
	}

	// 4. Otherwise we are on Linux -> get glibc version
	systemGlibc, err := getGlibcVersion()
	if err != nil {
		// If GLIBC detection fails, fallback to lowest GLIBC requirement
		return selectLowestGlibcBinary(binaries, currentArch), nil
	}

	// 5. Compare GLIBC versions -> select highest version that is <= system GLIBC version
	var bestBinary BorgBinary
	var bestGlibcVersion *version.Version

	for _, binary := range binaries {
		// Only consider binaries for current OS, architecture, and latest Borg version
		if binary.Os != currentOS || !binary.Version.Equal(latestVersion) {
			continue
		}

		// Check architecture compatibility (empty means any)
		if binary.Arch != "" && binary.Arch != currentArch {
			continue
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
		return selectLowestGlibcBinary(binaries, currentArch), nil
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

// selectLowestGlibcBinary returns the binary with the lowest GLIBC requirement for the given architecture
func selectLowestGlibcBinary(binaries []BorgBinary, arch string) BorgBinary {
	if len(binaries) == 0 {
		return BorgBinary{}
	}

	var lowest BorgBinary
	foundCompatible := false

	for _, binary := range binaries {
		// Check architecture compatibility (empty means any)
		if binary.Arch != "" && binary.Arch != arch {
			continue
		}

		if binary.GlibcVersion == nil {
			continue
		}

		if !foundCompatible || binary.GlibcVersion.LessThan(lowest.GlibcVersion) {
			lowest = binary
			foundCompatible = true
		}
	}

	return lowest
}

// GetMountBorgBinary selects the latest Borg binary that supports mount operations for the current system.
// On Linux, this will typically return the same as GetLatestBorgBinary since all Linux binaries support mount.
// On macOS, this will return the latest single binary (not -gh directory builds) which includes FUSE support.
func GetMountBorgBinary(binaries []BorgBinary) (BorgBinary, error) {
	currentOS := OS(runtime.GOOS)
	currentArch := runtime.GOARCH
	if !IsLinux() && !IsMacOS() {
		return BorgBinary{}, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	// Find latest binary for current OS and architecture that supports mount
	var latestBinary BorgBinary
	var latestVersion *version.Version

	for _, binary := range binaries {
		// Check OS compatibility
		if binary.Os != currentOS {
			continue
		}

		// Check architecture compatibility (empty means any)
		if binary.Arch != "" && binary.Arch != currentArch {
			continue
		}

		// Must support mount
		if !binary.SupportsMount {
			continue
		}

		// Track the latest version
		if latestVersion == nil || binary.Version.GreaterThan(latestVersion) {
			latestBinary = binary
			latestVersion = binary.Version
		}
	}

	if latestVersion == nil {
		return BorgBinary{}, fmt.Errorf("no mount-capable binary found for operating system %s and architecture %s", runtime.GOOS, currentArch)
	}

	// If on Darwin, return the architecture-matched binary
	if IsMacOS() {
		return latestBinary, nil
	}

	// Otherwise we are on Linux -> get glibc version and select appropriate binary
	systemGlibc, err := getGlibcVersion()
	if err != nil {
		// If GLIBC detection fails, fallback to lowest GLIBC requirement with mount support
		return selectLowestGlibcMountBinary(binaries, currentArch), nil
	}

	// Compare GLIBC versions -> select highest version that is <= system GLIBC version
	var bestBinary BorgBinary
	var bestGlibcVersion *version.Version

	for _, binary := range binaries {
		// Only consider binaries for current OS, latest Borg version, and mount support
		if binary.Os != currentOS || !binary.Version.Equal(latestVersion) || !binary.SupportsMount {
			continue
		}

		// Check architecture compatibility (empty means any)
		if binary.Arch != "" && binary.Arch != currentArch {
			continue
		}

		if binary.GlibcVersion != nil && binary.GlibcVersion.LessThanOrEqual(systemGlibc) {
			if bestGlibcVersion == nil || binary.GlibcVersion.GreaterThan(bestGlibcVersion) {
				bestBinary = binary
				bestGlibcVersion = binary.GlibcVersion
			}
		}
	}

	if bestGlibcVersion == nil {
		// No compatible GLIBC version found, return lowest available with mount support
		return selectLowestGlibcMountBinary(binaries, currentArch), nil
	}

	return bestBinary, nil
}

// selectLowestGlibcMountBinary returns the binary with the lowest GLIBC requirement that supports mount
func selectLowestGlibcMountBinary(binaries []BorgBinary, arch string) BorgBinary {
	if len(binaries) == 0 {
		return BorgBinary{}
	}

	var lowest BorgBinary
	foundCompatible := false

	for _, binary := range binaries {
		// Check architecture compatibility (empty means any)
		if binary.Arch != "" && binary.Arch != arch {
			continue
		}

		// Must support mount
		if !binary.SupportsMount {
			continue
		}

		if binary.GlibcVersion == nil {
			continue
		}

		if !foundCompatible || binary.GlibcVersion.LessThan(lowest.GlibcVersion) {
			lowest = binary
			foundCompatible = true
		}
	}

	return lowest
}
