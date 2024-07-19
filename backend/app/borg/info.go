package borg

import (
	"arco/backend/util"
	"fmt"
	"os/exec"
)

func (b *Borg) Info(url, password string) error {
	cmd := exec.Command(b.path, "info", "--json", url)
	cmd.Env = util.BorgEnv{}.WithPassword(password).AsList()

	// Check if we can connect to the repository
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	}
	b.log.LogCmdEnd(cmd.String(), startTime)
	return nil
}
