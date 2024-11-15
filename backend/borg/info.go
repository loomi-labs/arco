package borg

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type InfoResponse struct {
	Archives    []ArchiveInfo `json:"archives"`
	Cache       Cache         `json:"cache"`
	Encryption  Encryption    `json:"encryption"`
	Repository  Repository    `json:"repository"`
	SecurityDir string        `json:"security_dir"`
}

func (b *borg) Info(ctx context.Context, repository, password string) (*InfoResponse, error) {
	cmd := exec.CommandContext(ctx, b.path, "info", "--json", repository)
	cmd.Env = Env{}.WithPassword(password).AsList()

	// Check if we can connect to the repository
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, b.log.LogCmdError(cmd.String(), startTime, err)
	}

	var info InfoResponse
	err = json.Unmarshal(out, &info)
	if err != nil {
		return nil, b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("failed to parse borg info output: %w", err))
	}

	b.log.LogCmdEnd(cmd.String(), startTime)
	return &info, nil
}
