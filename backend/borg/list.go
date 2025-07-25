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

func (b *borg) List(ctx context.Context, repository string, password string) (*types.ListResponse, *types.Status) {
	cmd := exec.CommandContext(ctx, b.path, "list", "--json", "--format", "`{end}`", repository)
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	// Get the list from the borg repository
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()

	// Convert command output and error to Status
	status := combinedOutputToStatus(out, err)
	if status.HasError() {
		return nil, b.log.LogCmdStatus(ctx, status, cmd.String(), time.Since(startTime))
	}

	var listResponse types.ListResponse
	err = json.Unmarshal(utils.SanitizeOutput(out, b.log.SugaredLogger), &listResponse)
	if err != nil {
		parseStatus := newStatusWithError(fmt.Errorf("failed to parse borg list output: %v", err))
		return nil, b.log.LogCmdStatus(ctx, parseStatus, cmd.String(), time.Since(startTime))
	}

	return &listResponse, b.log.LogCmdStatus(ctx, status, cmd.String(), time.Since(startTime))
}
