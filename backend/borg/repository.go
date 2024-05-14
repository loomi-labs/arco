package borg

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"os/exec"
)

type Repo struct {
	Id         string `json:"id"`
	Url        string `json:"url"`
	binaryPath string
	log        logger.Logger
}

func NewRepo(log logger.Logger, binaryPath string) *Repo {
	return &Repo{
		log:        log,
		binaryPath: binaryPath,
		Id:         uuid.New().String(),
	}
}

func (r *Repo) Info() (*ListResponse, error) {
	cmd := exec.Command(r.binaryPath, "info", "--json", r.Url)
	cmd.Env = getEnv()
	r.log.Debug(fmt.Sprintf("Running command: %s", cmd.String()))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", out, err)
	}

	var listResponse ListResponse
	err = json.Unmarshal(out, &listResponse)
	if err != nil {
		return nil, err
	}

	return &listResponse, nil
}
