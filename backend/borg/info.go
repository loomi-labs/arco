package borg

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loomi-labs/arco/backend/borg/types"
	"github.com/loomi-labs/arco/backend/borg/utils"
	"os/exec"
	"time"
)

func (cr *commandRunner) Info(cmd *exec.Cmd) ([]byte, error) {
	return cmd.CombinedOutput()
}


func (b *borg) Info(ctx context.Context, repository, password string) (*types.InfoResponse, *types.Status) {
	cmd := exec.CommandContext(ctx, b.path, "info", "--json", repository)
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	// Check if we can connect to the repository
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := b.commandRunner.Info(cmd)

	// Convert command output and error to Status
	status := combinedOutputToStatus(out, err)
	if status.HasError() {
		return nil, b.log.LogCmdResult(ctx, status, cmd.String(), time.Since(startTime))
	}

	var info types.InfoResponse
	err = json.Unmarshal(utils.SanitizeOutput(out, b.log.SugaredLogger), &info)
	if err != nil {
		parseStatus := newStatusWithError(fmt.Errorf("failed to parse borg info output: %v", err))
		return nil, b.log.LogCmdResult(ctx, parseStatus, cmd.String(), time.Since(startTime))
	}

	return &info, b.log.LogCmdResult(ctx, status, cmd.String(), time.Since(startTime))
}
