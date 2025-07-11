package borg

import (
	"context"
	"os/exec"
	"time"
)

func (b *borg) Init(ctx context.Context, repository, password string, noPassword bool) *Status {
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

	return b.log.LogCmdResult(status, cmd.String(), time.Since(startTime))
}
