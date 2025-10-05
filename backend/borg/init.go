package borg

import (
	"context"
	"os/exec"
	"time"

	"github.com/loomi-labs/arco/backend/borg/types"
)

func (b *borg) Init(ctx context.Context, repository, password string, noPassword bool) *types.Status {
	cmdList := []string{"init"}
	if noPassword {
		cmdList = append(cmdList, "--encryption=none")
	} else {
		cmdList = append(cmdList, "--encryption=repokey-blake2")
	}
	cmdList = append(cmdList, repository)

	cmd := exec.CommandContext(ctx, b.path, cmdList...)
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	status := combinedOutputToStatus(out, err)

	return b.log.LogCmdStatus(ctx, status, cmd.String(), time.Since(startTime))
}
