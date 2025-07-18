package borg

import (
	"context"
	"fmt"
	"github.com/loomi-labs/arco/backend/borg/types"
	"os/exec"
	"time"
)

// Rename renames an archive in the repository.
func (b *borg) Rename(ctx context.Context, repository, archive, password, newName string) *types.Status {
	cmd := exec.CommandContext(ctx, b.path, "rename", fmt.Sprintf("%s::%s", repository, archive), newName)
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	status := combinedOutputToStatus(out, err)

	return b.log.LogCmdStatus(ctx, status, cmd.String(), time.Since(startTime))
}
