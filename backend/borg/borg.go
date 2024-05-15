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
	"timebender/backend/ent/backupprofile"
	"timebender/backend/ent/repository"
	"timebender/backend/ssh"
)

type Borg struct {
	binaryPath string
	log        *zap.SugaredLogger
	client     *ent.Client
}

func NewBorg(log *zap.SugaredLogger, client *ent.Client) *Borg {
	return &Borg{
		binaryPath: "bin/borg-linuxnewer64",
		log:        log,
		client:     client,
	}
}

func createEnv(password string) []string {
	sshOptions := []string{
		"-oBatchMode=yes",
		"-oStrictHostKeyChecking=accept-new",
		"-i ~/sshtest/id_storage_test",
	}
	env := append(
		os.Environ(),
		fmt.Sprintf("BORG_PASSPHRASE=%s", password),
		fmt.Sprintf("BORG_RSH=%s", fmt.Sprintf("ssh %s", strings.Join(sshOptions, " "))),
	)
	return env
}

func getEnv() []string {
	sshOptions := []string{
		"-oBatchMode=yes",
		"-oStrictHostKeyChecking=accept-new",
		"-i ~/sshtest/id_storage_test",
	}
	env := append(
		os.Environ(),
		fmt.Sprintf("BORG_RSH=%s", fmt.Sprintf("ssh %s", strings.Join(sshOptions, " "))),
	)
	return env
}

func getTestEnvOverride() []string {
	passphrase := os.Getenv("BORG_PASSPHRASE")
	env := append(
		getEnv(),
		fmt.Sprintf("BORG_PASSPHRASE=%s", passphrase),
		fmt.Sprintf("BORG_NEW_PASSPHRASE=%s", passphrase),
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
	return b.client.BackupProfile.
		Query().
		WithRepositories().
		Where(backupprofile.ID(id)).Only(context.Background())
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

/****************/
/* Repositories */
/****************/

func (b *Borg) GetRepository(id int) (*ent.Repository, error) {
	return b.client.Repository.
		Query().
		WithBackupprofiles().
		Where(repository.ID(id)).
		Only(context.Background())
}

func (b *Borg) GetRepositories() ([]*ent.Repository, error) {
	return b.client.Repository.Query().All(context.Background())
}

func (b *Borg) GetArchives() (*ListResponse, error) {
	repo, err := b.GetRepository(0)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(b.binaryPath, "list", "--json", repo.URL)
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

func (b *Borg) AddExistingRepository(name, url, password string, backupProfileId int) (*ent.Repository, error) {
	cmd := exec.Command(b.binaryPath, "info", "--json", url)
	cmd.Env = createEnv(password)
	b.log.Info(fmt.Sprintf("Running command: %s", cmd.String()))

	// Check if we can connect to the repository
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", out, err)
	}

	// Create a new repository entity
	return b.client.Repository.
		Create().
		SetName(name).
		SetURL(url).
		SetPassword(password).
		AddBackupprofileIDs(backupProfileId).
		Save(context.Background())
}
