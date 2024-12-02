package borg

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loomi-labs/arco/backend/borg/types"
	"os/exec"
)

func (cr *commandRunner) Info(cmd *exec.Cmd) ([]byte, error) {
	return cmd.CombinedOutput()
}

func (b *borg) Info(ctx context.Context, repository, password string) (*types.InfoResponse, error) {
	cmd := exec.CommandContext(ctx, b.path, "info", "--json", repository)
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	// Check if we can connect to the repository
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := b.commandRunner.Info(cmd)
	if err != nil {
		return nil, b.log.LogCmdError(ctx, cmd.String(), startTime, err)
	}

	var info types.InfoResponse
	err = json.Unmarshal(out, &info)
	if err != nil {
		return nil, b.log.LogCmdError(ctx, cmd.String(), startTime, fmt.Errorf("failed to parse borg info output: %w", err))
	}

	b.log.LogCmdEnd(cmd.String(), startTime)
	return &info, nil
}
