package borg

import (
	"fmt"
	"os/exec"
)

func (b *Borg) Init(url, password string, noPassword bool) error {
	cmdList := []string{"init"}
	if noPassword {
		cmdList = append(cmdList, "--encryption=none")
	} else {
		cmdList = append(cmdList, "--encryption=repokey-blake2")
	}
	cmdList = append(cmdList, url)

	cmd := exec.Command(b.path, cmdList...)
	cmd.Env = Env{}.WithPassword(password).AsList()

	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	}
	b.log.LogCmdEnd(cmd.String(), startTime)
	return nil
}
