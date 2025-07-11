package borg

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

// Compact runs the borg compact command to free up space in the repository
func (b *borg) Compact(ctx context.Context, repository string, password string) *Status {
	// Prepare compact command
	cmd := exec.CommandContext(ctx, b.path, "compact", repository)
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	// Add cancel functionality
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	}

	// Run compact command
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	status := combinedOutputToStatus(out, err)
	return b.log.LogCmdResult(status, cmd.String(), time.Since(startTime))
}
