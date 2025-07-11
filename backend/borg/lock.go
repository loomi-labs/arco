package borg

import (
	"context"
	"os/exec"
	"time"
)

// BreakLock deletes the lock for the given repository.
func (b *borg) BreakLock(ctx context.Context, repository string, password string) *Status {
	cmd := exec.CommandContext(ctx, b.path, "break-lock", repository)
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	status := combinedOutputToStatus(out, err)

	return b.log.LogCmdResult(status, cmd.String(), time.Since(startTime))
}
