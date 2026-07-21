package platform

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/hashicorp/go-version"
)

// Binaries contains all available Borg binary variants
var Binaries = []BorgBinary{
	// Borg 1.4.5 - Linux x86_64 (glibc231 build was dropped upstream in 1.4.5; min glibc is now 2.35)
	{
		Name:          "borg_1.4.5",
		Version:       version.Must(version.NewVersion("1.4.5")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.35")),
		Arch:          "amd64",
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.5/borg-linux-glibc235-x86_64-gh",
		SupportsMount: true,
	},
	// Borg 1.4.5 - Linux ARM64
	{
		Name:          "borg_1.4.5",
		Version:       version.Must(version.NewVersion("1.4.5")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.35")),
		Arch:          "arm64",
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.5/borg-linux-glibc235-arm64-gh",
		SupportsMount: true,
	},
	// Borg 1.4.5 - macOS Intel (directory distribution for faster startup, no FUSE support)
	{
		Name:          "borg_1.4.5",
		Version:       version.Must(version.NewVersion("1.4.5")),
		Os:            Darwin,
		MacOSVersion:  version.Must(version.NewVersion("15.0")), // built on macOS 15, requires macOS 15+ per upstream README
		Arch:          "amd64",
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.5/borg-macos-15-x86_64-gh.tgz",
		IsDirectory:   true,
		SupportsMount: false, // -gh builds don't include llfuse
	},
	// Borg 1.4.5 - macOS Apple Silicon (directory distribution for faster startup, no FUSE support)
	{
		Name:          "borg_1.4.5",
		Version:       version.Must(version.NewVersion("1.4.5")),
		Os:            Darwin,
		MacOSVersion:  version.Must(version.NewVersion("15.0")), // built on macOS 15, requires macOS 15+ per upstream README
		Arch:          "arm64",
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.5/borg-macos-15-arm64-gh.tgz",
		IsDirectory:   true,
		SupportsMount: false, // -gh builds don't include llfuse
	},
	// Borg 1.4.4 - Linux x86_64 variants
	{
		Name:          "borg_1.4.4",
		Version:       version.Must(version.NewVersion("1.4.4")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.31")),
		Arch:          "amd64",
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.4/borg-linux-glibc231-x86_64",
		SupportsMount: true,
	},
	{
		Name:          "borg_1.4.4",
		Version:       version.Must(version.NewVersion("1.4.4")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.35")),
		Arch:          "amd64",
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.4/borg-linux-glibc235-x86_64-gh",
		SupportsMount: true,
	},
	// Borg 1.4.4 - Linux ARM64
	{
		Name:          "borg_1.4.4",
		Version:       version.Must(version.NewVersion("1.4.4")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.35")),
		Arch:          "arm64",
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.4/borg-linux-glibc235-arm64-gh",
		SupportsMount: true,
	},
	// Borg 1.4.4 - macOS Intel (directory distribution for faster startup, no FUSE support)
	{
		Name:          "borg_1.4.4",
		Version:       version.Must(version.NewVersion("1.4.4")),
		Os:            Darwin,
		MacOSVersion:  version.Must(version.NewVersion("15.0")), // built on macOS 15, requires macOS 15+ per upstream README
		Arch:          "amd64",
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.4/borg-macos-15-x86_64-gh.tgz",
		IsDirectory:   true,
		SupportsMount: false, // -gh builds don't include llfuse
	},
	// Borg 1.4.4 - macOS Apple Silicon (directory distribution for faster startup, no FUSE support)
	{
		Name:          "borg_1.4.4",
		Version:       version.Must(version.NewVersion("1.4.4")),
		Os:            Darwin,
		MacOSVersion:  version.Must(version.NewVersion("15.0")), // built on macOS 15, requires macOS 15+ per upstream README
		Arch:          "arm64",
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.4/borg-macos-15-arm64-gh.tgz",
		IsDirectory:   true,
		SupportsMount: false, // -gh builds don't include llfuse
	},
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
		MacOSVersion:  version.Must(version.NewVersion("13.0")), // built on macOS 13
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
		MacOSVersion:  version.Must(version.NewVersion("14.0")), // built on macOS 14
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
		Arch:          "amd64", // upstream 1.4.1 fat binaries are x86_64-only (no official arm64 build)
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-linux-glibc228",
		SupportsMount: true,
	},
	{
		Name:          "borg_1.4.1",
		Version:       version.Must(version.NewVersion("1.4.1")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.31")),
		Arch:          "amd64", // upstream 1.4.1 fat binaries are x86_64-only (no official arm64 build)
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-linux-glibc231",
		SupportsMount: true,
	},
	{
		Name:          "borg_1.4.1",
		Version:       version.Must(version.NewVersion("1.4.1")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.36")),
		Arch:          "amd64", // upstream 1.4.1 fat binaries are x86_64-only (no official arm64 build)
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-linux-glibc236",
		SupportsMount: true,
	},
	// Borg 1.4.1 - macOS (directory distribution with FUSE support)
	{
		Name:          "borg_1.4.1",
		Version:       version.Must(version.NewVersion("1.4.1")),
		Os:            Darwin,
		MacOSVersion:  version.Must(version.NewVersion("10.12")), // built on macOS 10.12
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-macos1012.tgz",
		IsDirectory:   true,
		SupportsMount: true, // Non-gh builds include llfuse
	},
	// Borg 1.4.0 - Linux variants
	{
		Name:          "borg_1.4.0",
		Version:       version.Must(version.NewVersion("1.4.0")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.28")),
		Arch:          "amd64", // upstream 1.4.0 fat binaries are x86_64-only (no official arm64 build)
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-linux-glibc228",
		SupportsMount: true,
	},
	{
		Name:          "borg_1.4.0",
		Version:       version.Must(version.NewVersion("1.4.0")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.31")),
		Arch:          "amd64", // upstream 1.4.0 fat binaries are x86_64-only (no official arm64 build)
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-linux-glibc231",
		SupportsMount: true,
	},
	{
		Name:          "borg_1.4.0",
		Version:       version.Must(version.NewVersion("1.4.0")),
		Os:            Linux,
		GlibcVersion:  version.Must(version.NewVersion("2.36")),
		Arch:          "amd64", // upstream 1.4.0 fat binaries are x86_64-only (no official arm64 build)
		Url:           "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-linux-glibc236",
		SupportsMount: true,
	},
	// Borg 1.4.0 - macOS (single binary with FUSE support)
	{
		Name:          "borg_1.4.0",
		Version:       version.Must(version.NewVersion("1.4.0")),
		Os:            Darwin,
		MacOSVersion:  version.Must(version.NewVersion("10.12")), // built on macOS 10.12
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

	// 2. Ensure at least one binary exists for current OS and architecture
	found := false

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

		found = true
		break
	}

	if !found {
		return BorgBinary{}, fmt.Errorf("no binary found for operating system %s and architecture %s", runtime.GOOS, currentArch)
	}

	// 3. If on Darwin -> get macOS version and select appropriate binary
	if IsMacOS() {
		systemMacOS, err := getMacOSVersion()
		if err != nil {
			// If macOS version detection fails, fallback to lowest macOS requirement
			return selectLowestMacOSBinary(binaries, currentArch, false), nil
		}

		// Select the highest Borg version whose macOS requirement the system satisfies.
		// Like the glibc path below, this looks across all versions: 1.4.5/1.4.4 binaries
		// require macOS 15, so older hosts fall back to the newest still-compatible
		// version (e.g. 1.4.3) instead of getting a binary that won't run.
		return selectBestMacOSBinary(binaries, currentArch, systemMacOS, false), nil
	}

	// 4. Otherwise we are on Linux -> get glibc version
	systemGlibc, err := getGlibcVersion()
	if err != nil {
		// If GLIBC detection fails, fallback to lowest GLIBC requirement
		return selectLowestGlibcBinary(binaries, currentArch), nil
	}

	// 5. Select the highest Borg version whose glibc requirement the system satisfies.
	// This intentionally looks across all versions rather than only the newest one: upstream
	// occasionally drops a glibc build for the latest version (e.g. 1.4.5 dropped glibc231),
	// so a system on an older glibc should fall back to the newest still-compatible version
	// (1.4.4-glibc231) instead of the globally-oldest build.
	return selectBestGlibcBinary(binaries, currentArch, systemGlibc, false), nil
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

// getMacOSVersion detects the macOS product version on Darwin systems
func getMacOSVersion() (*version.Version, error) {
	if !IsMacOS() {
		return nil, fmt.Errorf("only macOS supports sw_vers") // Not applicable for non-Darwin systems
	}

	cmd := exec.Command("sw_vers", "-productVersion")
	// SYSTEM_VERSION_COMPAT=1 inherited from the environment makes sw_vers report "10.16"
	// on Big Sur and later; force it off so we always see the real product version.
	cmd.Env = append(os.Environ(), "SYSTEM_VERSION_COMPAT=0")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to detect macOS version: %w", err)
	}

	// Output is the bare product version, e.g. "15.5" or "13.6.1"
	versionCandidate := strings.TrimSpace(string(output))
	re := regexp.MustCompile(`^\d+(\.\d+)*$`)
	if !re.MatchString(versionCandidate) {
		return nil, fmt.Errorf("could not parse macOS version from sw_vers output: %s", versionCandidate)
	}

	v, err := version.NewVersion(versionCandidate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse macOS version %s: %w", versionCandidate, err)
	}

	// "10.16" never existed as a real product version; it is the compat shim's alias for
	// Big Sur and later. Treat it as a detection failure so selection falls back to the
	// lowest-requirement binary, which runs fine on any Big Sur+ machine.
	if v.Equal(version.Must(version.NewVersion("10.16"))) {
		return nil, fmt.Errorf("sw_vers reported compat-shimmed macOS version 10.16")
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

	// Ensure at least one mount-capable binary exists for current OS and architecture
	found := false

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

		found = true
		break
	}

	if !found {
		return BorgBinary{}, fmt.Errorf("no mount-capable binary found for operating system %s and architecture %s", runtime.GOOS, currentArch)
	}

	// If on Darwin -> get macOS version and select appropriate mount-capable binary
	if IsMacOS() {
		systemMacOS, err := getMacOSVersion()
		if err != nil {
			// If macOS version detection fails, fallback to lowest macOS requirement with mount support
			return selectLowestMacOSBinary(binaries, currentArch, true), nil
		}

		// Select the highest mount-capable Borg version whose macOS requirement the system satisfies.
		return selectBestMacOSBinary(binaries, currentArch, systemMacOS, true), nil
	}

	// Otherwise we are on Linux -> get glibc version and select appropriate binary
	systemGlibc, err := getGlibcVersion()
	if err != nil {
		// If GLIBC detection fails, fallback to lowest GLIBC requirement with mount support
		return selectLowestGlibcMountBinary(binaries, currentArch), nil
	}

	// Select the highest mount-capable Borg version whose glibc requirement the system satisfies.
	return selectBestGlibcBinary(binaries, currentArch, systemGlibc, true), nil
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

// selectBestGlibcBinary picks the best Linux binary for the given architecture and system glibc
// version. It returns the highest Borg version whose glibc requirement is satisfied by the system
// (GlibcVersion <= systemGlibc), breaking ties toward the higher glibc build. When mountOnly is
// set, only mount-capable binaries are considered.
//
// Unlike a "newest version, then filter by glibc" approach, this searches across all versions so
// that a system on an older glibc still gets the newest compatible version even when the latest
// release dropped a glibc build (e.g. 1.4.5 dropped the x86_64 glibc231 build). If the system
// glibc is older than every available build, it falls back to the lowest-glibc binary.
func selectBestGlibcBinary(binaries []BorgBinary, arch string, systemGlibc *version.Version, mountOnly bool) BorgBinary {
	var best BorgBinary
	found := false

	for _, binary := range binaries {
		if binary.Os != Linux {
			continue
		}

		// Check architecture compatibility (empty means any)
		if binary.Arch != "" && binary.Arch != arch {
			continue
		}

		if mountOnly && !binary.SupportsMount {
			continue
		}

		if binary.GlibcVersion == nil {
			continue
		}

		// Skip builds that require a newer glibc than the system provides
		if binary.GlibcVersion.GreaterThan(systemGlibc) {
			continue
		}

		if !found ||
			binary.Version.GreaterThan(best.Version) ||
			(binary.Version.Equal(best.Version) && binary.GlibcVersion.GreaterThan(best.GlibcVersion)) {
			best = binary
			found = true
		}
	}

	if !found {
		// System glibc is older than every available build; return the lowest-glibc option.
		if mountOnly {
			return selectLowestGlibcMountBinary(binaries, arch)
		}
		return selectLowestGlibcBinary(binaries, arch)
	}

	return best
}

// selectBestMacOSBinary picks the best Darwin binary for the given architecture and macOS
// version. It returns the highest Borg version whose macOS requirement is satisfied by the
// system (MacOSVersion <= systemMacOS, nil meaning compatible with any macOS), breaking ties
// toward the higher macOS requirement. When mountOnly is set, only mount-capable binaries are
// considered.
//
// Like selectBestGlibcBinary, this searches across all versions so that an older macOS still
// gets the newest compatible version (e.g. 1.4.5/1.4.4 require macOS 15, so a macOS 14 arm64
// host falls back to 1.4.3). If the system is older than every available build, it falls back
// to the lowest-requirement binary.
func selectBestMacOSBinary(binaries []BorgBinary, arch string, systemMacOS *version.Version, mountOnly bool) BorgBinary {
	var best BorgBinary
	found := false

	for _, binary := range binaries {
		if binary.Os != Darwin {
			continue
		}

		// Check architecture compatibility (empty means any)
		if binary.Arch != "" && binary.Arch != arch {
			continue
		}

		if mountOnly && !binary.SupportsMount {
			continue
		}

		// Skip builds that require a newer macOS than the system provides.
		// A nil MacOSVersion means the binary is compatible with any macOS.
		if binary.MacOSVersion != nil && binary.MacOSVersion.GreaterThan(systemMacOS) {
			continue
		}

		if !found ||
			binary.Version.GreaterThan(best.Version) ||
			(binary.Version.Equal(best.Version) && binary.MacOSVersion != nil &&
				(best.MacOSVersion == nil || binary.MacOSVersion.GreaterThan(best.MacOSVersion))) {
			best = binary
			found = true
		}
	}

	if !found {
		// System macOS is older than every available build; return the lowest-requirement option.
		return selectLowestMacOSBinary(binaries, arch, mountOnly)
	}

	return best
}

// selectLowestMacOSBinary returns the Darwin binary with the lowest macOS requirement for the
// given architecture (nil MacOSVersion counts as the lowest possible requirement). Ties on the
// requirement are broken toward the higher Borg version. When mountOnly is set, only
// mount-capable binaries are considered.
func selectLowestMacOSBinary(binaries []BorgBinary, arch string, mountOnly bool) BorgBinary {
	var lowest BorgBinary
	found := false

	for _, binary := range binaries {
		if binary.Os != Darwin {
			continue
		}

		// Check architecture compatibility (empty means any)
		if binary.Arch != "" && binary.Arch != arch {
			continue
		}

		if mountOnly && !binary.SupportsMount {
			continue
		}

		if !found || macOSRequirementLess(binary, lowest) ||
			(macOSRequirementEqual(binary, lowest) && binary.Version.GreaterThan(lowest.Version)) {
			lowest = binary
			found = true
		}
	}

	return lowest
}

// macOSRequirementLess reports whether a's macOS requirement is strictly lower than b's,
// treating nil as the lowest possible requirement.
func macOSRequirementLess(a, b BorgBinary) bool {
	if a.MacOSVersion == nil {
		return b.MacOSVersion != nil
	}
	if b.MacOSVersion == nil {
		return false
	}
	return a.MacOSVersion.LessThan(b.MacOSVersion)
}

// macOSRequirementEqual reports whether a and b have the same macOS requirement,
// treating two nil requirements as equal.
func macOSRequirementEqual(a, b BorgBinary) bool {
	if a.MacOSVersion == nil || b.MacOSVersion == nil {
		return a.MacOSVersion == b.MacOSVersion
	}
	return a.MacOSVersion.Equal(b.MacOSVersion)
}
