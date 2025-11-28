package borg

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/loomi-labs/arco/backend/borg/types"
)

// Recreate modifies an archive's comment using borg recreate.
func (b *borg) Recreate(ctx context.Context, repository, archive, password, comment string) *types.Status {
	cmd := exec.CommandContext(ctx, b.path, "recreate", "--comment", comment, fmt.Sprintf("%s::%s", repository, archive))
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	status := combinedOutputToStatus(out, err)

	return b.log.LogCmdStatus(ctx, status, cmd.String(), time.Since(startTime))
}
