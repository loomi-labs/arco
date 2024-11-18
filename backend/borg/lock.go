package borg

import (
	"context"
	"fmt"
	"os/exec"
)

// BreakLock deletes the lock for the given repository.
func (b *borg) BreakLock(ctx context.Context, repository string, password string) error {
	cmd := exec.CommandContext(ctx, b.path, "break-lock", repository)
	cmd.Env = Env{}.WithPassword(password).AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return b.log.LogCmdError(ctx, cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	}
	b.log.LogCmdEnd(cmd.String(), startTime)
	return nil
}
