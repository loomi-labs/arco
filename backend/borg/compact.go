package borg

import (
	"context"
	"github.com/loomi-labs/arco/backend/borg/types"
	"os/exec"
	"syscall"
	"time"
)

// Compact runs the borg compact command to free up space in the repository
func (b *borg) Compact(ctx context.Context, repository string, password string) *types.Status {
	// Prepare compact command
	cmd := exec.CommandContext(ctx, b.path, "compact", repository)
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	// Add cancel functionality
	hasBeenCanceled := false
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		hasBeenCanceled = true
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	}

	// Run compact command
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()

	if hasBeenCanceled {
		// We don't care about the real status of the borg operation because we canceled it
		status := newStatusWithCanceled()
		return b.log.LogCmdResult(ctx, status, cmd.String(), time.Since(startTime))
	}

	status := combinedOutputToStatus(out, err)
	return b.log.LogCmdStatus(ctx, status, cmd.String(), time.Since(startTime))
}
