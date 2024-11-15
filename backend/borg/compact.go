package borg

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
)

// Compact runs the borg compact command to free up space in the repository
func (b *borg) Compact(ctx context.Context, repository string, password string) error {
	// Prepare compact command
	cmd := exec.CommandContext(ctx, b.path, "compact", repository)
	cmd.Env = Env{}.WithPassword(password).AsList()

	// Add cancel functionality
	hasBeenCanceled := false
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		hasBeenCanceled = true
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
	}

	// Run compact command
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		if hasBeenCanceled {
			b.log.LogCmdCancelled(cmd.String(), startTime)
			return CancelErr{}
		}
		return b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	} else {
		b.log.LogCmdEnd(cmd.String(), startTime)
	}
	return nil
}
