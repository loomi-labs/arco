package borg

import (
	"context"
	"fmt"
	"github.com/loomi-labs/arco/backend/borg/types"
	"os/exec"
	"time"
)

func (b *borg) MountRepository(ctx context.Context, repository string, password string, mountPath string) *types.Status {
	return b.mount(ctx, repository, nil, password, mountPath)
}

func (b *borg) MountArchive(ctx context.Context, repository string, archive string, password string, mountPath string) *types.Status {
	return b.mount(ctx, repository, &archive, password, mountPath)
}

func (b *borg) mount(ctx context.Context, repository string, archive *string, password string, mountPath string) *types.Status {
	archiveOrRepo := repository
	if archive != nil {
		archiveOrRepo = fmt.Sprintf("%s::%s", repository, *archive)
	}

	cmd := exec.CommandContext(ctx, b.path, "mount", archiveOrRepo, mountPath)
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	status := combinedOutputToStatus(out, err)

	return b.log.LogCmdResult(ctx, status, cmd.String(), time.Since(startTime))
}

func (b *borg) Umount(ctx context.Context, path string) *types.Status {
	cmd := exec.CommandContext(ctx, b.path, "umount", path)

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	status := combinedOutputToStatus(out, err)

	return b.log.LogCmdResult(ctx, status, cmd.String(), time.Since(startTime))
}
