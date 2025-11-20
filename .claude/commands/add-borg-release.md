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

### 1. Add Binary Definitions

**File:** `backend/platform/borg.go`

Add new binary definitions for each platform variant. Follow the existing pattern around lines 14-75.

For version X.Y.Z, add:

```go
// Linux variants (3 glibc versions)
{
    Name:         "borg_X.Y.Z",
    Version:      version.Must(version.NewVersion("X.Y.Z")),
    Os:           Linux,
    GlibcVersion: version.Must(version.NewVersion("2.28")),
    Url:          "https://github.com/borgbackup/borg/releases/download/X.Y.Z/borg-linux-glibc228",
},
{
    Name:         "borg_X.Y.Z",
    Version:      version.Must(version.NewVersion("X.Y.Z")),
    Os:           Linux,
    GlibcVersion: version.Must(version.NewVersion("2.31")),
    Url:          "https://github.com/borgbackup/borg/releases/download/X.Y.Z/borg-linux-glibc231",
},
{
    Name:         "borg_X.Y.Z",
    Version:      version.Must(version.NewVersion("X.Y.Z")),
    Os:           Linux,
    GlibcVersion: version.Must(version.NewVersion("2.36")),
    Url:          "https://github.com/borgbackup/borg/releases/download/X.Y.Z/borg-linux-glibc236",
},
// macOS variant
{
    Name:         "borg_X.Y.Z",
    Version:      version.Must(version.NewVersion("X.Y.Z")),
    Os:           Darwin,
    GlibcVersion: nil,
    Url:          "https://github.com/borgbackup/borg/releases/download/X.Y.Z/borg-macos1012",
},
```

**Important:** Verify the exact binary names on the GitHub release page, as they may change between versions.

### 2. Update Default Version

**File:** `backend/cmd/root.go`

Update the default version to the new release at lines ~151-153:

```go
BorgBinaries: platform.Binaries,
BorgPath:     filepath.Join(configDir, "borg_X.Y.Z"),  // Update version
BorgVersion:  "X.Y.Z",  // Update version
```

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

- The application automatically detects the best binary for the system (based on OS and GLIBC version)
- `GetLatestBorgBinary()` in `backend/platform/borg.go` handles automatic selection
- Multiple versions can coexist; the app uses semantic versioning to find the latest
- Old versions can remain in the code for backwards compatibility and testing

## Common Issues

1. **Binary URLs incorrect:** Always verify exact binary names on GitHub release page
2. **Version parsing fails:** Ensure version strings follow semantic versioning (X.Y.Z)
3. **macOS binary name changed:** Check if Apple still uses "macos1012" identifier or if it's been updated