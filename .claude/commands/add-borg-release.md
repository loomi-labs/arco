# Add New Borg Release

Add a new Borg backup release version to the Arco application. This command guides you through all necessary code changes.

## Prerequisites

Before starting, verify:
1. The new Borg release exists on GitHub: https://github.com/borgbackup/borg/releases
2. You have the version number (e.g., "1.4.2")
3. The release includes binaries for:
   - Linux: `borg-linux-glibc228`, `borg-linux-glibc231`, `borg-linux-glibc236`
   - macOS: `borg-macos1012` (or updated macOS version identifier)

## Step-by-Step Instructions

### 0. Verify Binary Names (CRITICAL)

Borg's binary naming convention has changed over time. Always verify actual filenames before adding binary definitions.

**Method 1: GitHub API (Recommended)**

```bash
curl -s "https://api.github.com/repos/borgbackup/borg/releases/tags/X.Y.Z" | \
  grep '"name":' | grep -E 'borg-(linux|macos)' | \
  sed 's/.*"name": "\(.*\)".*/\1/' | sort
```

**Method 2: Check release page directly**

Visit: `https://github.com/borgbackup/borg/releases/tag/X.Y.Z` and look at the Assets section.

**Known naming patterns:**
- **Borg 1.4.0-1.4.1:**
  - Linux: `borg-linux-glibc228`, `borg-linux-glibc231`, `borg-linux-glibc236`
  - macOS: `borg-macos1012` (universal binary)
- **Borg 1.4.2+:**
  - Linux: `borg-linux-glibc231-x86_64`, `borg-linux-glibc235-x86_64-gh`
  - macOS: `borg-macos-13-x86_64-gh` (Intel), `borg-macos-14-arm64-gh` (Apple Silicon)

### 1. Add Binary Definitions

**File:** `backend/platform/borg.go`

Add new binary definitions for each platform variant. Follow the existing pattern around lines 14-110.

**IMPORTANT:** Use the binary names from Step 0. The template below shows both legacy (pre-1.4.2) and modern (1.4.2+) patterns.

#### For Modern Releases (1.4.2+) with Architecture-Specific Binaries:

```go
// Borg X.Y.Z - Linux variants (check actual glibc versions from Step 0)
{
    Name:         "borg_X.Y.Z",
    Version:      version.Must(version.NewVersion("X.Y.Z")),
    Os:           Linux,
    GlibcVersion: version.Must(version.NewVersion("2.31")),  // Adjust based on actual binaries
    Arch:         "amd64",  // Required for architecture-specific binaries
    Url:          "https://github.com/borgbackup/borg/releases/download/X.Y.Z/borg-linux-glibc231-x86_64",
},
{
    Name:         "borg_X.Y.Z",
    Version:      version.Must(version.NewVersion("X.Y.Z")),
    Os:           Linux,
    GlibcVersion: version.Must(version.NewVersion("2.35")),
    Arch:         "amd64",
    Url:          "https://github.com/borgbackup/borg/releases/download/X.Y.Z/borg-linux-glibc235-x86_64-gh",
},
// Borg X.Y.Z - macOS Intel (x86_64)
{
    Name:         "borg_X.Y.Z",
    Version:      version.Must(version.NewVersion("X.Y.Z")),
    Os:           Darwin,
    GlibcVersion: nil,
    Arch:         "amd64",  // Intel Macs
    Url:          "https://github.com/borgbackup/borg/releases/download/X.Y.Z/borg-macos-13-x86_64-gh",
},
// Borg X.Y.Z - macOS Apple Silicon (ARM64)
{
    Name:         "borg_X.Y.Z",
    Version:      version.Must(version.NewVersion("X.Y.Z")),
    Os:           Darwin,
    GlibcVersion: nil,
    Arch:         "arm64",  // Apple Silicon
    Url:          "https://github.com/borgbackup/borg/releases/download/X.Y.Z/borg-macos-14-arm64-gh",
},
```

#### For Legacy Releases (pre-1.4.2) with Universal Binaries:

```go
// Borg X.Y.Z - Linux variants
{
    Name:         "borg_X.Y.Z",
    Version:      version.Must(version.NewVersion("X.Y.Z")),
    Os:           Linux,
    GlibcVersion: version.Must(version.NewVersion("2.28")),
    Arch:         "",  // Empty = works on any architecture
    Url:          "https://github.com/borgbackup/borg/releases/download/X.Y.Z/borg-linux-glibc228",
},
// ... repeat for glibc 2.31, 2.36
// Borg X.Y.Z - macOS (universal)
{
    Name:         "borg_X.Y.Z",
    Version:      version.Must(version.NewVersion("X.Y.Z")),
    Os:           Darwin,
    GlibcVersion: nil,
    Arch:         "",  // Empty = works on both Intel and ARM64 via Rosetta 2
    Url:          "https://github.com/borgbackup/borg/releases/download/X.Y.Z/borg-macos1012",
},
```

**Key Points:**
- Set `Arch` field to `"amd64"` or `"arm64"` for architecture-specific binaries
- Leave `Arch` empty (`""`) for universal binaries that work on any architecture
- macOS universal binaries (pre-1.4.2) work on both Intel and Apple Silicon via Rosetta 2
- Always verify exact binary names from Step 0 - they change between versions!

### 2. Verify Default Version (Automatic)

**File:** `backend/cmd/root.go`

The default Borg version is **automatically set** to use `platform.Binaries[0]` (lines ~152-153).

Since you added the new version binaries at the **top** of the `Binaries` array in Step 1, the new version is automatically the default. **No code changes needed in this file.**

```go
// This code automatically uses the first binary in the array:
BorgPath:    filepath.Join(configDir, platform.Binaries[0].Name),
BorgVersion: platform.Binaries[0].Version.String(),
```

**Only update this manually if** you want to use a specific older version as default (not recommended).

### 3. Update Docker Files

Update Docker configurations to use the new version:

#### Client Dockerfiles
**Files:**
- `docker/borg-client/ubuntu-20.04.Dockerfile`
- `docker/borg-client/ubuntu-22.04.Dockerfile`
- `docker/borg-client/ubuntu-24.04.Dockerfile`

Update the default `CLIENT_BORG_VERSION` and `SERVER_BORG_VERSION` ARG values (first occurrence in each file):

```dockerfile
ARG CLIENT_BORG_VERSION="X.Y.Z"
ARG SERVER_BORG_VERSION="X.Y.Z"
```

#### Server Dockerfile
**File:** `docker/borg-server/Dockerfile`

Update the default `BORG_VERSION` ARG value:

```dockerfile
ARG BORG_VERSION="X.Y.Z"
```

### 4. Update CI/CD Workflows

**File:** `.github/workflows/integration_tests.yml`

Add the new version to the CI test matrix strategy around lines 35-36:

```yaml
matrix:
  borg-version: ["1.4.0", "1.4.1", "X.Y.Z"]  # Add new version
```

### 5. Update Test Scripts

**File:** `scripts/run-integration-test.sh`

Add usage examples in comments to document the new version:

```bash
# Example usage with new version:
# ./scripts/run-integration-test.sh --client-version X.Y.Z --server-version X.Y.Z
```

## Architecture Detection (Since 1.4.2)

Starting with Borg 1.4.2, architecture-specific binaries are supported for better platform compatibility.

### How It Works

**BorgBinary.Arch Field:**
- Set to `"amd64"` for x86_64/Intel binaries
- Set to `"arm64"` for ARM64/Apple Silicon binaries
- Leave empty (`""`) for universal binaries (backward compatibility)

**Automatic Selection:**
- `GetLatestBorgBinary()` in `backend/platform/borg.go` automatically filters by `runtime.GOARCH`
- macOS: Automatically selects correct binary for Intel or Apple Silicon
- Linux: Currently supports x86_64 (amd64); ARM64 binaries available for 1.4.2+ if needed

**Backward Compatibility:**
- Binaries with empty `Arch` field work on any architecture
- Older versions (1.4.0, 1.4.1) use universal binaries
- macOS universal binaries run on Apple Silicon via Rosetta 2

### 6. Review Release Notes for Compatibility

**URL:** `https://github.com/borgbackup/borg/releases/tag/X.Y.Z`

Thoroughly review the Borg release notes for:

1. **Breaking changes:**
   - Command-line argument changes
   - Configuration file format changes
   - Repository format changes (may require migration)
   - API or behavior changes

2. **Deprecated features:**
   - Features that may be removed in future versions
   - Alternative approaches we should adopt

3. **New features:**
   - New flags or options we might want to expose in Arco
   - Performance improvements we can leverage
   - Security enhancements

4. **Bug fixes:**
   - Issues that may have affected Arco users
   - Fixes that may change expected behavior

5. **System requirements:**
   - Changes to minimum GLIBC versions (update binary variants if needed)
   - New dependencies or system requirements
   - Platform support changes

**Action items:**
- Document any required code changes in a checklist
- Update Arco's code if the new version introduces breaking changes
- Consider adding new Arco features that leverage new Borg capabilities
- Update documentation if behavior changes affect users

## Verification

After making changes:

1. **Format code:**
   ```bash
   NO_COLOR=1 task dev:format
   ```

2. **Build the application:**
   ```bash
   NO_COLOR=1 task build
   ```

3. **Test locally:**
   - Run the app and verify it downloads the correct binary
   - Check the config directory for `borg_X.Y.Z` binary
   - Verify the version with: `~/.config/arco/borg_X.Y.Z --version`

4. **Run integration tests:**
   ```bash
   ./scripts/run-integration-test.sh --client-version X.Y.Z --server-version X.Y.Z
   ```

## Notes

- The application automatically detects the best binary for the system (based on OS, architecture, and GLIBC version)
- `GetLatestBorgBinary()` in `backend/platform/borg.go` handles automatic selection
- **Architecture-specific binaries** (1.4.2+): The `Arch` field enables per-architecture selection
- **Backward compatibility**: Empty `Arch` field means the binary works on any architecture
- **macOS architecture detection**: Automatically selects Intel (amd64) or Apple Silicon (arm64) binary based on system
- Multiple versions can coexist; the app uses semantic versioning to find the latest
- Old versions can remain in the code for backwards compatibility and testing

## Common Issues

1. **Binary URLs incorrect:** Always verify exact binary names on GitHub release page (use Step 0 method)
2. **Version parsing fails:** Ensure version strings follow semantic versioning (X.Y.Z)
3. **macOS binary name changed:** Borg 1.4.2+ uses architecture-specific naming (see Step 0 for patterns)
4. **Architecture mismatch:** If binaries are architecture-specific, ensure `Arch` field is set correctly:
   - Use GitHub API (Step 0) to check if binaries include architecture in filename (e.g., `-x86_64`, `-arm64`)
   - Set `Arch: "amd64"` for x86_64/Intel binaries
   - Set `Arch: "arm64"` for ARM64/Apple Silicon binaries
   - Leave `Arch: ""` (empty string) for universal binaries
5. **Binary download fails (404):** Binary naming changed in 1.4.2 - verify exact names with Step 0 before adding definitions