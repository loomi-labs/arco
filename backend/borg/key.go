package borg

import (
	"context"
	"os/exec"
	"time"

	"github.com/loomi-labs/arco/backend/borg/types"
)

// ChangePassphrase changes the passphrase of the repository's encryption key.
func (b *borg) ChangePassphrase(ctx context.Context, repository, currentPassword, newPassword string) *types.Status {
	cmd := exec.CommandContext(ctx, b.path, "key", "change-passphrase", repository)
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(currentPassword).WithNewPassword(newPassword).AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	status := combinedOutputToStatus(out, err)

	return b.log.LogCmdStatus(ctx, status, cmd.String(), time.Since(startTime))
}
