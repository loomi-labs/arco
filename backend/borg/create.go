package borg

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// Create creates a new backup in the repository.
// It is long running and should be run in a goroutine.
func (b *Borg) Create(ctx context.Context, repoUrl, password, prefix string, directories []string) error {
	// Prepare backup command
	name := fmt.Sprintf("%s-%s", prefix, time.Now().In(time.Local).Format("2006-01-02-15-04-05"))
	cmd := exec.CommandContext(ctx, b.path, append([]string{
		"create",
		fmt.Sprintf("%s::%s", repoUrl, name)},
		directories...,
	)...)
	cmd.Env = Env{}.WithPassword(password).AsList()

	// Run backup command
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	} else {
		b.log.LogCmdEnd(cmd.String(), startTime)
	}
	return nil
}
