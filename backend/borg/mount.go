package borg

import (
	"context"
	"fmt"
	"os/exec"
)

func (b *borg) MountRepository(ctx context.Context, repository string, password string, mountPath string) error {
	return b.mount(ctx, repository, nil, password, mountPath)
}

func (b *borg) MountArchive(ctx context.Context, repository string, archive string, password string, mountPath string) error {
	return b.mount(ctx, repository, &archive, password, mountPath)
}

func (b *borg) mount(ctx context.Context, repository string, archive *string, password string, mountPath string) error {
	archiveOrRepo := repository
	if archive != nil {
		archiveOrRepo = fmt.Sprintf("%s::%s", repository, *archive)
	}

	cmd := exec.CommandContext(ctx, b.path, "mount", archiveOrRepo, mountPath)
	cmd.Env = Env{}.WithPassword(password).AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	}
	b.log.LogCmdEnd(cmd.String(), startTime)
	return nil
}

func (b *borg) Umount(ctx context.Context, path string) error {
	cmd := exec.CommandContext(ctx, b.path, "umount", path)

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	}
	b.log.LogCmdEnd(cmd.String(), startTime)
	return nil
}
