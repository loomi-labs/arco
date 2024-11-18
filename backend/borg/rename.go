package borg

import (
	"context"
	"fmt"
	"os/exec"
)

// Rename renames an archive in the repository.
func (b *borg) Rename(ctx context.Context, repository, archive, password, newName string) error {
	cmd := exec.CommandContext(ctx, b.path, "rename", fmt.Sprintf("%s::%s", repository, archive), newName)
	cmd.Env = Env{}.WithPassword(password).AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return b.log.LogCmdError(ctx, cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	}
	b.log.LogCmdEnd(cmd.String(), startTime)
	return nil
}
