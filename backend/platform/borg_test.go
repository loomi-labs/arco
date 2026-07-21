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
			// arm64 has no glibc231 build; a 2.29 arm64 system falls back across versions to
			// the newest build whose glibc fits (1.4.1-glibc228, a universal/empty-Arch build).
			name:        "glibc 2.29 arm64 -> newest compatible universal fallback",
			arch:        "arm64",
			systemGlibc: "2.29",
			wantName:    "borg_1.4.1",
			wantURL:     "https://github.com/borgbackup/borg/releases/download/1.4.1/borg-linux-glibc228",
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
		})
	}
}
