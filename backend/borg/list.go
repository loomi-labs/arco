package borg

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loomi-labs/arco/backend/borg/types"
	"os/exec"
)

func (b *borg) List(ctx context.Context, repository string, password string) (*types.ListResponse, error) {
	cmd := exec.CommandContext(ctx, b.path, "list", "--json", "--format", "`{end}`", repository)
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	// Get the list from the borg repository
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, b.log.LogCmdError(ctx, cmd.String(), startTime, err)
	}

	var listResponse types.ListResponse
	err = json.Unmarshal(out, &listResponse)
	if err != nil {
		return nil, b.log.LogCmdError(ctx, cmd.String(), startTime, fmt.Errorf("failed to parse borg list output: %w", err))
	}

	b.log.LogCmdEnd(cmd.String(), startTime)
	return &listResponse, nil
}
