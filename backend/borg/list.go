package borg

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type ListResponse struct {
	Archives   []ArchiveList `json:"archives"`
	Encryption Encryption    `json:"encryption"`
	Repository Repository    `json:"repository"`
}

func (b *borg) List(repoUrl string, password string) (*ListResponse, error) {
	cmd := exec.Command(b.path, "list", "--json", "--format", "`{end}`", repoUrl)
	cmd.Env = Env{}.WithPassword(password).AsList()

	// Get the list from the borg repository
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, b.log.LogCmdError(cmd.String(), startTime, err)
	}

	var listResponse ListResponse
	err = json.Unmarshal(out, &listResponse)
	if err != nil {
		return nil, b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("failed to parse borg list output: %w", err))
	}

	b.log.LogCmdEnd(cmd.String(), startTime)
	return &listResponse, nil
}
