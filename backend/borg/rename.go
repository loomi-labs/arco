package borg

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// Rename renames an archive in the repository.
func (b *borg) Rename(ctx context.Context, repository, archive, password, newName string) *Status {
	cmd := exec.CommandContext(ctx, b.path, "rename", fmt.Sprintf("%s::%s", repository, archive), newName)
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	result := combinedOutputToStatus(out, err)

	return b.log.LogCmdResult(ctx, result, cmd.String(), time.Since(startTime))
}
