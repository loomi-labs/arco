package borg

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"strings"
	"timebender/backend/ent"
	"timebender/backend/ssh"
)

type Borg struct {
	binaryPath     string
	log            *zap.SugaredLogger
	client         *ent.Client
	backupProfiles []ent.BackupProfile
	repositories   []Repo
}

func NewBorg(log *zap.SugaredLogger, client *ent.Client) *Borg {
	return &Borg{
		binaryPath: "bin/borg-linuxnewer64",
		log:        log,
		client:     client,
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

func (b *Borg) HandleError(msg string, fErr *FrontendError) {
	errStr := ""
	if fErr != nil {
		if fErr.Message != "" && fErr.Stack != "" {
			errStr = fmt.Sprintf("%s\n%s", fErr.Message, fErr.Stack)
		} else if fErr.Message != "" {
			errStr = fErr.Message
		}
	}

	// We don't want to show the stack trace from the go code because the error comes from the frontend
	b.log.WithOptions(zap.AddCallerSkip(9999999)).
		Errorf(fmt.Sprintf("%s: %s", msg, errStr))
}

/******************/
/* Backup Profile */
/******************/

func (b *Borg) NewBackupProfile() (*ent.BackupProfile, error) {
	hostname, _ := os.Hostname()
	return b.client.BackupProfile.Create().
		SetName(hostname).
		SetPrefix(hostname).
		SetDirectories([]string{}).
		SetHasPeriodicBackups(true).
		//SetPeriodicBackupTime(time.Date(0, 0, 0, 9, 0, 0, 0, time.Local)).
		Save(context.Background())
}

func (b *Borg) GetDirectorySuggestions() []string {
	home, _ := os.UserHomeDir()
	if home != "" {
		return []string{home}
	}
	return []string{}
}

func (b *Borg) GetBackupProfile(id int) (*ent.BackupProfile, error) {
	return b.client.BackupProfile.Get(context.Background(), id)
}

func (b *Borg) GetBackupProfiles() ([]*ent.BackupProfile, error) {
	return b.client.BackupProfile.Query().All(context.Background())
}

func (b *Borg) SaveBackupProfile(backup ent.BackupProfile) error {
	_, err := b.client.BackupProfile.
		UpdateOneID(backup.ID).
		SetName(backup.Name).
		SetPrefix(backup.Prefix).
		SetDirectories(backup.Directories).
		SetHasPeriodicBackups(backup.HasPeriodicBackups).
		//SetPeriodicBackupTime(backup.PeriodicBackupTime).
		SetIsSetupComplete(backup.IsSetupComplete).
		Save(context.Background())
	return err
}

/***************/
/* Backup Sets */
/***************/

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
