package borg

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func (c *Client) Version() (string, error) {
	cmd := exec.Command(c.binaryPath, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *Client) Backup() error {
	root := os.Getenv("BORG_ROOT")
	repo := os.Getenv("BORG_REPO")
	path := os.Getenv("BORG_BACKUP_PATHS")

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	name := fmt.Sprintf("%s-%s", hostname, time.Now().In(time.Local).Format("2006-01-02-15-04-05"))

	cmd := exec.Command(c.binaryPath, "create", fmt.Sprintf("%s%s::%s", root, repo, name), path)
	cmd.Env = getEnv()
	c.log.Info(fmt.Sprintf("Running command: %s", cmd.String()))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", out, err)
	}

	return nil
}

func (c *Client) Prune() error {
	repo := os.Getenv("BORG_REPO")

	cmd := exec.Command(c.binaryPath, "prune", "--list", repo, "--keep-daily", "7")
	cmd.Env = getEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", out, err)
	}

	return nil
}

func (c *Client) InitRepo(repoName string) error {
	root := os.Getenv("BORG_ROOT")
	repo := fmt.Sprintf("%s/~/%s", root, repoName)

	quota := "10G"

	cmd := exec.Command(c.binaryPath, "init", "--encryption=repokey-blake2", "--storage-quota", quota, repo)

	// log command
	c.log.Info(fmt.Sprintf("Running command: %s", cmd.String()))

	cmd.Env = getEnv()
	out, err := cmd.CombinedOutput()
	c.log.Info(fmt.Sprintf("Output: %s", out))
	if err != nil {
		return fmt.Errorf("%s: %s", out, err)
	}

	return nil
}
