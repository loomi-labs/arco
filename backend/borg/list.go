package borg

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type Archive struct {
	Archive  string `json:"archive"`
	Barchive string `json:"barchive"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Start    string `json:"start"`
	Time     string `json:"time"`
}

type Encryption struct {
	Mode string `json:"mode"`
}

type Repository struct {
	ID           string `json:"id"`
	LastModified string `json:"last_modified"`
	Location     string `json:"location"`
}

type ListResponse struct {
	Archives   []Archive  `json:"archives"`
	Encryption Encryption `json:"encryption"`
	Repository Repository `json:"repository"`
}

func (b *Borg) List(repoUrl string, password string) (*ListResponse, error) {
	cmd := exec.Command(b.path, "list", "--json", repoUrl)
	cmd.Env = Env{}.WithPassword(password).AsList()

	// Get the list from the borg repository
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, b.log.LogCmdError(cmd.String(), startTime, err)
	}
	b.log.LogCmdEnd(cmd.String(), startTime)

	var listResponse ListResponse
	err = json.Unmarshal(out, &listResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %s, error: %w", out, err)
	}

	return &listResponse, nil
}
