package borg

import (
	"encoding/json"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"os"
	"os/exec"
	"time"
)

type Borg struct {
	binaryPath string
	log        logger.Logger
}

func NewBorg(log logger.Logger) *Borg {
	return &Borg{
		binaryPath: "bin/borg-linuxnewer64",
		log:        log,
	}
}

func (b *Borg) Version() (string, error) {
	cmd := exec.Command(b.binaryPath, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (b *Borg) List() (*ListResponse, error) {
	repo := os.Getenv("BORG_REPO")
	passphrase := os.Getenv("BORG_PASSPHRASE")
	b.log.Debug(fmt.Sprintf("Listing repo: %s", repo))

	cmd := exec.Command(b.binaryPath, "list", "--json", repo)
	cmd.Env = append(os.Environ(), fmt.Sprintf("BORG_PASSPHRASE=%s", passphrase))
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

func (b *Borg) Backup() error {
	repo := os.Getenv("BORG_REPO")
	passphrase := os.Getenv("BORG_PASSPHRASE")
	path := os.Getenv("BORG_BACKUP_PATHS")

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	name := fmt.Sprintf("%s-%s", hostname, time.Now().In(time.Local).Format("2006-01-02-15-04-05"))

	cmd := exec.Command(b.binaryPath, "create", fmt.Sprintf("%s::%s", repo, name), path)
	cmd.Env = append(os.Environ(), fmt.Sprintf("BORG_PASSPHRASE=%s", passphrase))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", out, err)
	}

	return nil
}
