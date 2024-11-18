package borg

import (
	"context"
	"fmt"
	"os/exec"
)

func (b *borg) Init(ctx context.Context, repository, password string, noPassword bool) error {
	cmdList := []string{"init"}
	if noPassword {
		cmdList = append(cmdList, "--encryption=none")
	} else {
		cmdList = append(cmdList, "--encryption=repokey-blake2")
	}
	cmdList = append(cmdList, repository)

	cmd := exec.CommandContext(ctx, b.path, cmdList...)
	cmd.Env = Env{}.WithPassword(password).AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return b.log.LogCmdError(ctx, cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	}
	b.log.LogCmdEnd(cmd.String(), startTime)
	return nil
}
