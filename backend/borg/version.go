package borg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/loomi-labs/arco/backend/borg/types"
)

// Version returns the main borg binary version
func (b *borg) Version(ctx context.Context) (*version.Version, *types.Status) {
	return b.versionAt(ctx, b.path)
}

// MountVersion returns the mount borg binary version
func (b *borg) MountVersion(ctx context.Context) (*version.Version, *types.Status) {
	return b.versionAt(ctx, b.mountPath)
}

func (b *borg) versionAt(ctx context.Context, path string) (*version.Version, *types.Status) {
	cmd := exec.CommandContext(ctx, path, "--version")
	startTime := b.log.LogCmdStart(cmd.String())

	out, err := cmd.CombinedOutput()
	status := combinedOutputToStatus(out, err)
	status = b.log.LogCmdStatus(ctx, status, cmd.String(), time.Since(startTime))

	if status.HasError() {
		return nil, status
	}

	// Output format: "borg 1.4.3\n" -> extract "1.4.3"
	fields := strings.Fields(string(out))
	if len(fields) < 2 {
		parseStatus := newStatusWithError(fmt.Errorf("unexpected version output: %s", string(out)))
		return nil, parseStatus
	}

	v, err := version.NewVersion(fields[1])
	if err != nil {
		parseStatus := newStatusWithError(fmt.Errorf("failed to parse version %q: %w", fields[1], err))
		return nil, parseStatus
	}

	return v, status
}
