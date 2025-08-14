package platform

import (
	"fmt"
	"runtime"

	"github.com/hashicorp/go-version"
	"github.com/loomi-labs/arco/backend/util"
)

// Binaries contains all available Borg binary variants
var Binaries = []BorgBinary{
	// Borg 1.4.1 - Linux variants
	{
		Name:         "borg_1.4.1_glibc228",
		Version:      version.Must(version.NewVersion("1.4.1")),
		Os:           util.Linux,
		GlibcVersion: version.Must(version.NewVersion("2.28")),
		Url:          "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-linux-glibc228",
	},
	{
		Name:         "borg_1.4.1_glibc231",
		Version:      version.Must(version.NewVersion("1.4.1")),
		Os:           util.Linux,
		GlibcVersion: version.Must(version.NewVersion("2.31")),
		Url:          "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-linux-glibc231",
	},
	{
		Name:         "borg_1.4.1_glibc236",
		Version:      version.Must(version.NewVersion("1.4.1")),
		Os:           util.Linux,
		GlibcVersion: version.Must(version.NewVersion("2.36")),
		Url:          "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-linux-glibc236",
	},
	// Borg 1.4.1 - macOS
	{
		Name:         "borg_1.4.1_macos",
		Version:      version.Must(version.NewVersion("1.4.1")),
		Os:           util.Darwin,
		GlibcVersion: nil,
		Url:          "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-macos1012",
	},
	// Borg 1.4.0 - Linux variants
	{
		Name:         "borg_1.4.0_glibc228",
		Version:      version.Must(version.NewVersion("1.4.0")),
		Os:           util.Linux,
		GlibcVersion: version.Must(version.NewVersion("2.28")),
		Url:          "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-linux-glibc228",
	},
	{
		Name:         "borg_1.4.0_glibc231",
		Version:      version.Must(version.NewVersion("1.4.0")),
		Os:           util.Linux,
		GlibcVersion: version.Must(version.NewVersion("2.31")),
		Url:          "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-linux-glibc231",
	},
	{
		Name:         "borg_1.4.0_glibc236",
		Version:      version.Must(version.NewVersion("1.4.0")),
		Os:           util.Linux,
		GlibcVersion: version.Must(version.NewVersion("2.36")),
		Url:          "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-linux-glibc236",
	},
	// Borg 1.4.0 - macOS
	{
		Name:         "borg_1.4.0_macos",
		Version:      version.Must(version.NewVersion("1.4.0")),
		Os:           util.Darwin,
		GlibcVersion: nil,
		Url:          "https://github.com/borgbackup/borg/releases/download/1.4.0/borg-macos1012",
	},
}

// GetLatestBorgBinary selects the appropriate Borg binary for the current system
func GetLatestBorgBinary(binaries []BorgBinary) (BorgBinary, error) {
	// 1. Check if Linux or Darwin -> if not return error
	currentOS := util.OS(runtime.GOOS)
	if !util.IsLinux() && !util.IsMacOS() {
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
	if util.IsMacOS() {
		return latestBinary, nil
	}

	// 4. Otherwise we are on Linux -> get glibc version
	systemGlibc, err := GetGlibcVersion()
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