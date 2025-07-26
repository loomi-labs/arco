package borg

import (
	"context"
	"github.com/loomi-labs/arco/backend/borg/types"
	"os/exec"
	"time"
)

// BreakLock deletes the lock for the given repository.
func (b *borg) BreakLock(ctx context.Context, repository string, password string) *types.Status {
	cmd := exec.CommandContext(ctx, b.path, "break-lock", repository)
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	status := combinedOutputToStatus(out, err)

	return b.log.LogCmdStatus(ctx, status, cmd.String(), time.Since(startTime))
}
