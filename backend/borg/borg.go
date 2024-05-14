package borg

import (
	"encoding/json"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"os"
	"os/exec"
	"strings"
	"time"
	"timebender/backend/ssh"
)

type Borg struct {
	binaryPath string
	log        logger.Logger
	BackupSets []BackupSet
}

func NewBorg(log logger.Logger) *Borg {
	return &Borg{
		binaryPath: "bin/borg-linuxnewer64",
		log:        log,
	}
}

func getEnv() []string {
	passphrase := os.Getenv("BORG_PASSPHRASE")
	sshOptions := []string{
		"-oBatchMode=yes",
		"-oStrictHostKeyChecking=accept-new",
		"-i /tmp/ssh/id_storage_test",
	}
	env := append(
		os.Environ(),
		fmt.Sprintf("BORG_PASSPHRASE=%s", passphrase),
		fmt.Sprintf("BORG_NEW_PASSPHRASE=%s", passphrase),
		fmt.Sprintf("BORG_RSH=%s", fmt.Sprintf("ssh %s", strings.Join(sshOptions, " "))),
	)
	return env
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
	repo := fmt.Sprintf("%s%s", os.Getenv("BORG_ROOT"), os.Getenv("BORG_REPO"))
	b.log.Debug(fmt.Sprintf("Listing repo: %s", repo))

	cmd := exec.Command(b.binaryPath, "list", "--json", repo)
	cmd.Env = getEnv()
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
	b.log.Debug(fmt.Sprintf("Running command: %s", cmd.String()))
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
	b.log.Debug(fmt.Sprintf("Running command: %s", cmd.String()))

	cmd.Env = getEnv()
	out, err := cmd.CombinedOutput()
	b.log.Debug(fmt.Sprintf("Output: %s", out))
	if err != nil {
		return fmt.Errorf("%s: %s", out, err)
	}

	return nil
}

func (b *Borg) CreateSSHKeyPair() (string, error) {
	pair, err := ssh.GenerateKeyPair()
	if err != nil {
		return "", err
	}
	b.log.Debug(fmt.Sprintf("Generated SSH key pair: %s", pair.AuthorizedKey()))
	return pair.AuthorizedKey(), nil
}

// ------------------------------------

func (b *Borg) NewBackupSet() *BackupSet {
	hostname, _ := os.Hostname()
	home, _ := os.UserHomeDir()
	return NewBackupSet(hostname, hostname, []string{home})
}

func (b *Borg) SaveBackupSet(backupSet *BackupSet) {
	// Add the backup-set to the list of backup-sets
	// If it already exists, update it
	for i, r := range b.BackupSets {
		if r.Id == backupSet.Id {
			b.BackupSets[i] = *backupSet
			return
		}
	}
	b.BackupSets = append(b.BackupSets, *backupSet)
}

func (b *Borg) GetBackupSet(id string) (*BackupSet, error) {
	for _, backupSet := range b.BackupSets {
		if backupSet.Id == id {
			return &backupSet, nil
		}
	}
	return nil, fmt.Errorf("backupSet with id %s not found", id)
}

func (b *Borg) ConnectExistingRepo() (*Repo, error) {
	repo := NewRepo(b.log, b.binaryPath)
	repo.Url = fmt.Sprintf("%s%s", os.Getenv("BORG_ROOT"), os.Getenv("BORG_REPO"))
	info, err := repo.Info()
	if err != nil {
		return nil, err
	}
	b.log.Debug(fmt.Sprintf("Connected to repo: %s", info))
	return repo, nil
}

//func (b *Borg) NewRepo() *Repo {
//	hostname, _ := os.Hostname()
//	home, _ := os.UserHomeDir()
//	return NewRepo(hostname, hostname, []string{home})
//}
//
//func (b *Borg) SaveRepo(repo *Repo) {
//	// Add the repo to the list of repositories
//	// If it already exists, update it
//	for i, r := range b.Repositories {
//		if r.Id == repo.Id {
//			b.Repositories[i] = *repo
//			return
//		}
//	}
//	b.Repositories = append(b.Repositories, *repo)
//}
//
//func (b *Borg) GetRepo(id string) (*Repo, error) {
//	b.log.Debug(fmt.Sprintf("Looking for repo with id: %s", id))
//	for _, repo := range b.Repositories {
//		if repo.Id == id {
//			b.log.Debug(fmt.Sprintf("Found repo: %s", repo.Name))
//			return &repo, nil
//		}
//	}
//	b.log.Error(fmt.Sprintf("Repo with id %s not found", id))
//	return nil, fmt.Errorf("repo with id %s not found", id)
//}
