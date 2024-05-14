package borg

import (
	"encoding/json"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"os"
	"os/exec"
	"strings"
	"timebender/backend/ssh"
)

type Borg struct {
	binaryPath   string
	log          logger.Logger
	backupSets   []BackupSet
	repositories []Repo
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
		"-i ~/sshtest/id_storage_test",
	}
	env := append(
		os.Environ(),
		fmt.Sprintf("BORG_PASSPHRASE=%s", passphrase),
		fmt.Sprintf("BORG_NEW_PASSPHRASE=%s", passphrase),
		fmt.Sprintf("BORG_RSH=%s", fmt.Sprintf("ssh %s", strings.Join(sshOptions, " "))),
	)
	return env
}

func (b *Borg) CreateSSHKeyPair() (string, error) {
	pair, err := ssh.GenerateKeyPair()
	if err != nil {
		return "", err
	}
	b.log.Debug(fmt.Sprintf("Generated SSH key pair: %s", pair.AuthorizedKey()))
	return pair.AuthorizedKey(), nil
}

/***************/
/* Backup Sets */
/***************/

func (b *Borg) NewBackupSet() *BackupSet {
	hostname, _ := os.Hostname()
	home, _ := os.UserHomeDir()
	backupSet := NewBackupSet(hostname, hostname, []string{home})
	b.backupSets = append(b.backupSets, *backupSet)
	return backupSet
}

func (b *Borg) GetBackupSet(id string) (*BackupSet, error) {
	for _, backupSet := range b.backupSets {
		if backupSet.Id == id {
			return &backupSet, nil
		}
	}
	return nil, fmt.Errorf("backupSet with id %s not found", id)
}

func (b *Borg) GetBackupSets() []BackupSet {
	return b.backupSets
}

func (b *Borg) AddDirectory(backupId string, newDir Directory) error {
	backup, err := b.GetBackupSet(backupId)
	if err != nil {
		return err
	}

	// Add directory to the list of directories
	// If it already exists, set IsAdded to true
	for i, dir := range backup.Directories {
		if dir.Path == newDir.Path {
			backup.Directories[i].IsAdded = true
			return nil
		}
	}
	backup.Directories = append(backup.Directories, newDir)
	return nil
}

//func (b *Borg) SaveBackupSet(backupSet *BackupSet) {
//	// Add the backup-set to the list of backup-sets
//	// If it already exists, update it
//	for i, r := range b.backupSets {
//		if r.Id == backupSet.Id {
//			b.backupSets[i] = *backupSet
//			return
//		}
//	}
//	b.backupSets = append(b.backupSets, *backupSet)
//}

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
//	for i, r := range b.repositories {
//		if r.Id == repo.Id {
//			b.repositories[i] = *repo
//			return
//		}
//	}
//	b.repositories = append(b.repositories, *repo)
//}
//
//func (b *Borg) GetRepo(id string) (*Repo, error) {
//	b.log.Debug(fmt.Sprintf("Looking for repo with id: %s", id))
//	for _, repo := range b.repositories {
//		if repo.Id == id {
//			b.log.Debug(fmt.Sprintf("Found repo: %s", repo.Name))
//			return &repo, nil
//		}
//	}
//	b.log.Error(fmt.Sprintf("Repo with id %s not found", id))
//	return nil, fmt.Errorf("repo with id %s not found", id)
//}

/****************/
/* Repositories */
/****************/

func (b *Borg) GetRepository(id string) (*Repo, error) {
	repo := NewRepo(b.log, b.binaryPath)
	repo.Url = fmt.Sprintf("%s%s", os.Getenv("BORG_ROOT"), os.Getenv("BORG_REPO"))
	info, err := repo.Info()
	if err != nil {
		return nil, err
	}
	b.log.Debug(fmt.Sprintf("Connected to repo: %s", info))
	return repo, nil
}

func (b *Borg) GetRepositories() ([]Repo, error) {
	repo := NewRepo(b.log, b.binaryPath)
	repo.Url = fmt.Sprintf("%s%s", os.Getenv("BORG_ROOT"), os.Getenv("BORG_REPO"))
	info, err := repo.Info()
	if err != nil {
		return nil, err
	}
	b.log.Debug(fmt.Sprintf("Connected to repo: %s", info))
	return []Repo{*repo}, nil
}

func (b *Borg) GetArchives() (*ListResponse, error) {
	repo, err := b.GetRepository("")
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(b.binaryPath, "list", "--json", repo.Url)
	cmd.Env = getEnv()
	b.log.Info(fmt.Sprintf("Running command: %s", cmd.String()))

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
