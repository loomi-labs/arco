package borg

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// DeleteArchive deletes a single archive from the repository
func (b *borg) DeleteArchive(ctx context.Context, repository string, archive string, password string) *Status {
	cmd := exec.CommandContext(ctx, b.path, "delete", fmt.Sprintf("%s::%s", repository, archive))
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	status := combinedOutputToStatus(out, err)

	return b.log.LogCmdResult(ctx, status, cmd.String(), time.Since(startTime))
}

// DeleteArchives deletes all archives with the given prefix from the repository.
// It is long running and should be run in a goroutine.
func (b *borg) DeleteArchives(ctx context.Context, repository, password, prefix string) *Status {
	// Prepare delete command
	cmd := exec.CommandContext(ctx, b.path,
		"delete",
		"--glob-archives",
		fmt.Sprintf("%s*", prefix),
		repository,
	)
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	// Run delete command
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	result := combinedOutputToStatus(out, err)

	if result.IsCompletedWithSuccess() {
		// Run compact to free up space
		compactResult := b.Compact(ctx, repository, password)
		if compactResult.HasError() {
			b.log.Errorf("Failed to compact after delete: %v", compactResult.GetError())
		}
	}

	return b.log.LogCmdResult(ctx, result, cmd.String(), time.Since(startTime))
}

// DeleteRepository deletes the repository and all its archives
func (b *borg) DeleteRepository(ctx context.Context, repository string, password string) *Status {
	cmd := exec.CommandContext(ctx, b.path, "delete", repository)
	cmd.Env = NewEnv(b.sshPrivateKeys).
		WithPassword(password).
		WithDeleteConfirmation().
		AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	status := combinedOutputToStatus(out, err)

	return b.log.LogCmdResult(ctx, status, cmd.String(), time.Since(startTime))
}
