package borg

import (
	"context"
	"fmt"
	"os/exec"
)

// DeleteArchive deletes a single archive from the repository
func (b *borg) DeleteArchive(ctx context.Context, repository string, archive string, password string) error {
	cmd := exec.CommandContext(ctx, b.path, "delete", fmt.Sprintf("%s::%s", repository, archive))
	cmd.Env = Env{}.WithPassword(password).AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return b.log.LogCmdError(ctx, cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	}
	b.log.LogCmdEnd(cmd.String(), startTime)
	return nil
}

// DeleteArchives deletes all archives with the given prefix from the repository.
// It is long running and should be run in a goroutine.
func (b *borg) DeleteArchives(ctx context.Context, repository, password, prefix string) error {
	// Prepare delete command
	cmd := exec.CommandContext(ctx, b.path,
		"delete",
		"--glob-archives",
		fmt.Sprintf("%s*", prefix),
		repository,
	)
	cmd.Env = Env{}.WithPassword(password).AsList()

	// Run delete command
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return b.log.LogCmdError(ctx, cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	} else {
		b.log.LogCmdEnd(cmd.String(), startTime)

		// Run compact to free up space
		return b.Compact(ctx, repository, password)
	}
}

// DeleteRepository deletes the repository and all its archives
func (b *borg) DeleteRepository(ctx context.Context, repository string, password string) error {
	cmd := exec.CommandContext(ctx, b.path, "delete", repository)
	cmd.Env = Env{}.
		WithPassword(password).
		WithDeleteConfirmation().
		AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return b.log.LogCmdError(ctx, cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	}
	b.log.LogCmdEnd(cmd.String(), startTime)
	return nil
}
