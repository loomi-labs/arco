package borg

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func (b *Borg) Version() (string, error) {
	cmd := exec.Command(b.binaryPath, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (b *Borg) Backup() error {
	root := os.Getenv("BORG_ROOT")
	repo := os.Getenv("BORG_REPO")
	path := os.Getenv("BORG_BACKUP_PATHS")

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	name := fmt.Sprintf("%s-%s", hostname, time.Now().In(time.Local).Format("2006-01-02-15-04-05"))

	cmd := exec.Command(b.binaryPath, "create", fmt.Sprintf("%s%s::%s", root, repo, name), path)
	cmd.Env = getEnv()
	b.log.Info(fmt.Sprintf("Running command: %s", cmd.String()))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", out, err)
	}

	return nil
}

func (b *Borg) Prune() error {
	repo := os.Getenv("BORG_REPO")

	cmd := exec.Command(b.binaryPath, "prune", "--list", repo, "--keep-daily", "7")
	cmd.Env = getEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", out, err)
	}

	return nil
}

func (b *Borg) InitRepo(repoName string) error {
	root := os.Getenv("BORG_ROOT")
	repo := fmt.Sprintf("%s/~/%s", root, repoName)

	quota := "10G"

	cmd := exec.Command(b.binaryPath, "init", "--encryption=repokey-blake2", "--storage-quota", quota, repo)

	// log command
	b.log.Info(fmt.Sprintf("Running command: %s", cmd.String()))

	cmd.Env = getEnv()
	out, err := cmd.CombinedOutput()
	b.log.Info(fmt.Sprintf("Output: %s", out))
	if err != nil {
		return fmt.Errorf("%s: %s", out, err)
	}

	return nil
}
