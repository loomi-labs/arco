package borg

import (
	"arco/backend/types"
	"context"
	"fmt"
	"os/exec"
)

// Delete multiple archives from the repository
// It is long running and should be run in a goroutine.
func (b *Borg) Delete(ctx context.Context, deleteJob types.DeleteJob) error {
	// Prepare delete command
	cmd := exec.CommandContext(ctx, b.path,
		"delete",
		"--glob-archives",
		fmt.Sprintf("%s-*", deleteJob.Prefix),
		deleteJob.RepoUrl,
	)
	cmd.Env = Env{}.WithPassword(deleteJob.RepoPassword).AsList()

	// Run delete command
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	} else {
		b.log.LogCmdEnd(cmd.String(), startTime)

		// Run compact to free up space
		return b.Compact(ctx, deleteJob.RepoUrl, deleteJob.RepoPassword)
	}
}

// DeleteArchive deletes a single archive from the repository
func (b *Borg) DeleteArchive(repository string, archive string, password string) error {
	cmd := exec.Command(b.path, "delete", fmt.Sprintf("%s::%s", repository, archive))
	cmd.Env = Env{}.WithPassword(password).AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	}
	b.log.LogCmdEnd(cmd.String(), startTime)
	return nil
}
