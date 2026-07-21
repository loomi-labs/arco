package platform

import (
	"testing"

	"github.com/hashicorp/go-version"
)

func mustVersion(t *testing.T, s string) *version.Version {
	t.Helper()
	v, err := version.NewVersion(s)
	if err != nil {
		t.Fatalf("failed to parse version %q: %v", s, err)
	}
	return v
}

// TestSelectBestGlibcBinary verifies that the newest Borg version compatible with the
// system glibc is selected, including the fallback when the latest release dropped a
// glibc build (1.4.5 dropped the x86_64 glibc231 build).
func TestSelectBestGlibcBinary(t *testing.T) {
	tests := []struct {
		name        string
		arch        string
		systemGlibc string
		mountOnly   bool
		wantName    string
		wantURL     string
	}{
		{
			name:        "modern glibc amd64 -> 1.4.5 glibc235",
			arch:        "amd64",
			systemGlibc: "2.40",
			wantName:    "borg_1.4.5",
			wantURL:     "https://github.com/borgbackup/borg/releases/download/1.4.5/borg-linux-glibc235-x86_64-gh",
		},
		{
			name:        "exactly glibc 2.35 amd64 -> 1.4.5 glibc235",
			arch:        "amd64",
			systemGlibc: "2.35",
			wantName:    "borg_1.4.5",
			wantURL:     "https://github.com/borgbackup/borg/releases/download/1.4.5/borg-linux-glibc235-x86_64-gh",
		},
		{
			// The regression fix: 1.4.5 has no glibc231 build, so a glibc 2.33 system
			// must fall back to the newest still-compatible version (1.4.4-glibc231),
			// NOT the globally-oldest build (1.4.1-glibc228).
			name:        "glibc 2.33 amd64 -> 1.4.4 glibc231 (regression fix)",
			arch:        "amd64",
			systemGlibc: "2.33",
			wantName:    "borg_1.4.4",
			wantURL:     "https://github.com/borgbackup/borg/releases/download/1.4.4/borg-linux-glibc231-x86_64",
		},
		{
			// Older than every glibc231/235 build -> lowest-glibc fallback (glibc228).
			name:        "glibc 2.29 amd64 -> lowest glibc fallback",
			arch:        "amd64",
			systemGlibc: "2.29",
			wantName:    "borg_1.4.1",
			wantURL:     "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-linux-glibc228",
		},
		{
			name:        "modern glibc arm64 -> 1.4.5 glibc235 arm64",
			arch:        "arm64",
			systemGlibc: "2.40",
			wantName:    "borg_1.4.5",
			wantURL:     "https://github.com/borgbackup/borg/releases/download/1.4.5/borg-linux-glibc235-arm64-gh",
		},
		{
			// No arm64 build exists below glibc 2.35, and the old fat binaries are x86_64-only.
			// An arm64 host must never be handed an x86_64 binary: it falls back to the
			// lowest-glibc arm64 build rather than a glibc228 x86_64 one.
			name:        "glibc 2.29 arm64 -> arm64 build only, never x86_64 fallback",
			arch:        "arm64",
			systemGlibc: "2.29",
			wantName:    "borg_1.4.5",
			wantURL:     "https://github.com/borgbackup/borg/releases/download/1.4.5/borg-linux-glibc235-arm64-gh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectBestGlibcBinary(Binaries, tt.arch, mustVersion(t, tt.systemGlibc), tt.mountOnly)
			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}
			if got.Url != tt.wantURL {
				t.Errorf("Url = %q, want %q", got.Url, tt.wantURL)
			}
			// A binary with an explicit Arch must never be selected for a different arch
			// (e.g. an arm64 host must not receive an x86_64-only build).
			if got.Arch != "" && got.Arch != tt.arch {
				t.Errorf("selected Arch = %q for %q host", got.Arch, tt.arch)
			}
		})
	}
}

// TestSelectBestMacOSBinary verifies that the newest Borg version compatible with the
// host macOS version is selected: 1.4.5/1.4.4 binaries are built on macOS 15 and require
// macOS 15+, so older hosts must fall back to the newest still-compatible version.
func TestSelectBestMacOSBinary(t *testing.T) {
	tests := []struct {
		name        string
		arch        string
		systemMacOS string
		mountOnly   bool
		wantName    string
		wantURL     string
	}{
		{
			name:        "macOS 15 amd64 -> 1.4.5",
			arch:        "amd64",
			systemMacOS: "15.5",
			wantName:    "borg_1.4.5",
			wantURL:     "https://github.com/borgbackup/borg/releases/download/1.4.5/borg-macos-15-x86_64-gh.tgz",
		},
		{
			name:        "exactly macOS 15.0 arm64 -> 1.4.5",
			arch:        "arm64",
			systemMacOS: "15.0",
			wantName:    "borg_1.4.5",
			wantURL:     "https://github.com/borgbackup/borg/releases/download/1.4.5/borg-macos-15-arm64-gh.tgz",
		},
		{
			// 1.4.5/1.4.4 require macOS 15, so a macOS 14 host must fall back to 1.4.3.
			name:        "macOS 14 arm64 -> 1.4.3",
			arch:        "arm64",
			systemMacOS: "14.7",
			wantName:    "borg_1.4.3",
			wantURL:     "https://github.com/borgbackup/borg/releases/download/1.4.3/borg-macos-14-arm64-gh.tgz",
		},
		{
			name:        "macOS 13 amd64 -> 1.4.3",
			arch:        "amd64",
			systemMacOS: "13.6",
			wantName:    "borg_1.4.3",
			wantURL:     "https://github.com/borgbackup/borg/releases/download/1.4.3/borg-macos-13-x86_64-gh.tgz",
		},
		{
			// The 1.4.3 arm64 build requires macOS 14, so a macOS 13 arm64 host must skip
			// past it down to the arch-less 1.4.1 build (empty Arch matches any arch).
			name:        "macOS 13 arm64 -> 1.4.1 (1.4.3 arm64 needs macOS 14)",
			arch:        "arm64",
			systemMacOS: "13.6",
			wantName:    "borg_1.4.1",
			wantURL:     "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-macos1012.tgz",
		},
		{
			name:        "macOS 12 amd64 -> 1.4.1",
			arch:        "amd64",
			systemMacOS: "12.7",
			wantName:    "borg_1.4.1",
			wantURL:     "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-macos1012.tgz",
		},
		{
			// Only the non-gh 1.4.1/1.4.0 builds include llfuse; 1.4.1 is the newest.
			name:        "mountOnly on macOS 15 -> 1.4.1",
			arch:        "arm64",
			systemMacOS: "15.5",
			mountOnly:   true,
			wantName:    "borg_1.4.1",
			wantURL:     "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-macos1012.tgz",
		},
		{
			// Older than every build -> lowest-requirement fallback; 1.4.1 and 1.4.0 both
			// require 10.12, so the tie must break toward the higher Borg version.
			name:        "macOS 10.11 -> lowest requirement fallback",
			arch:        "amd64",
			systemMacOS: "10.11",
			wantName:    "borg_1.4.1",
			wantURL:     "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-macos1012.tgz",
		},
		{
			// macOS jumped from 15 to 26 (Tahoe); plain numeric comparison must still work.
			name:        "macOS 26 arm64 -> 1.4.5",
			arch:        "arm64",
			systemMacOS: "26.0",
			wantName:    "borg_1.4.5",
			wantURL:     "https://github.com/borgbackup/borg/releases/download/1.4.5/borg-macos-15-arm64-gh.tgz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectBestMacOSBinary(Binaries, tt.arch, mustVersion(t, tt.systemMacOS), tt.mountOnly)
			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}
			if got.Url != tt.wantURL {
				t.Errorf("Url = %q, want %q", got.Url, tt.wantURL)
			}
			// A binary with an explicit Arch must never be selected for a different arch.
			if got.Arch != "" && got.Arch != tt.arch {
				t.Errorf("selected Arch = %q for %q host", got.Arch, tt.arch)
			}
		})
	}
}
