package borg

import (
	"context"
	"fmt"
	"github.com/labstack/gommon/log"
	"os/exec"
)

// Compact runs the borg compact command to free up space in the repository
func (b *Borg) Compact(ctx context.Context, repoUrl string, repoPassword string) error {
	// Prepare compact command
	cmd := exec.CommandContext(ctx, b.path, "compact", repoUrl)
	cmd.Env = Env{}.WithPassword(repoPassword).AsList()
	log.Debug("Command: ", cmd.String())

	// Run compact command
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	} else {
		b.log.LogCmdEnd(cmd.String(), startTime)
	}
	return nil
}
