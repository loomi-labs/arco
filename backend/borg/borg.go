package borg

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"
	"timebender/backend/ent"
	"timebender/backend/ent/archive"
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

func (b *Borg) createSSHKeyPair() (string, error) {
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

func (b *Borg) RunBackups(backupProfileId int) error {
	backupProfile, err := b.GetBackupProfile(backupProfileId)
	if err != nil {
		return err
	}
	if !backupProfile.IsSetupComplete {
		return fmt.Errorf("backup profile is not setup")
	}

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	// TODO: run this async
	for _, repo := range backupProfile.Edges.Repositories {
		name := fmt.Sprintf("%s-%s", hostname, time.Now().In(time.Local).Format("2006-01-02-15-04-05"))

		cmd := exec.Command(b.binaryPath, "create", fmt.Sprintf("%s::%s", repo.URL, name), strings.Join(backupProfile.Directories, " "))
		cmd.Env = createEnv(repo.Password)

		startTime := time.Now()
		b.log.Debug(fmt.Sprintf("Running command: %s", cmd.String()))
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s: %s", out, err)
		}
		b.log.Debug(fmt.Sprintf("Command took %s", time.Since(startTime)))
	}
	return nil
}

/****************/
/* Repositories */
/****************/

func (b *Borg) GetRepository(id int) (*ent.Repository, error) {
	return b.client.Repository.
		Query().
		WithBackupprofiles().
		WithArchives().
		Where(repository.ID(id)).
		Only(context.Background())
}

func (b *Borg) GetRepositories() ([]*ent.Repository, error) {
	return b.client.Repository.Query().All(context.Background())
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

func (b *Borg) RefreshArchives(repoId int) ([]*ent.Archive, error) {
	repo, err := b.GetRepository(repoId)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(b.binaryPath, "list", "--json", repo.URL)
	cmd.Env = createEnv(repo.Password)

	// Get the list from the borg repository
	startTime := time.Now()
	b.log.Debug(fmt.Sprintf("Running command: %s", cmd.String()))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", out, err)
	}
	b.log.Debug(fmt.Sprintf("Command took %s", time.Since(startTime)))

	var listResponse ListResponse
	err = json.Unmarshal(out, &listResponse)
	if err != nil {
		return nil, err
	}

	// Get all the borg ids
	borgIds := make([]string, len(listResponse.Archives))
	for i, arch := range listResponse.Archives {
		borgIds[i] = arch.ID
	}

	// Delete the archives that don't exist anymore
	cnt, err := b.client.Archive.
		Delete().
		Where(
			archive.And(
				archive.HasRepositoryWith(repository.ID(repoId)),
				archive.BorgIDNotIn(borgIds...),
			)).
		Exec(context.Background())
	if err != nil {
		return nil, err
	}
	if cnt > 0 {
		b.log.Debug(fmt.Sprintf("Deleted %d archives", cnt))
	}

	// Check which archives are already saved
	archives, err := b.client.Archive.
		Query().
		Where(archive.HasRepositoryWith(repository.ID(repoId))).
		All(context.Background())
	if err != nil {
		return nil, err
	}
	savedBorgIds := make([]string, len(archives))
	for i, arch := range archives {
		savedBorgIds[i] = arch.BorgID
	}

	// Save the new archives
	for _, arch := range listResponse.Archives {
		if !slices.Contains(savedBorgIds, arch.ID) {
			createdAt, err := time.Parse("2006-01-02T15:04:05.000000", arch.Start)
			if err != nil {
				return nil, err
			}
			duration, err := time.Parse("2006-01-02T15:04:05.000000", arch.Time)
			if err != nil {
				return nil, err
			}
			newArchive, err := b.client.Archive.
				Create().
				SetBorgID(arch.ID).
				SetName(arch.Name).
				SetCreatedAt(createdAt).
				SetDuration(duration).
				SetRepositoryID(repoId).
				Save(context.Background())
			if err != nil {
				return nil, err
			}
			archives = append(archives, newArchive)
			b.log.Debug(fmt.Sprintf("Saved archive: %s", newArchive.Name))
		}
	}

	return archives, nil
}

func (b *Borg) DeleteArchive(id int) error {
	arch, err := b.client.Archive.
		Query().
		WithRepository().
		Where(archive.ID(id)).
		Only(context.Background())
	if err != nil {
		return err
	}

	cmd := exec.Command(b.binaryPath, "delete", fmt.Sprintf("%s::%s", arch.Edges.Repository.URL, arch.Name))
	cmd.Env = createEnv(arch.Edges.Repository.Password)

	startTime := time.Now()
	b.log.Debug(fmt.Sprintf("Running command: %s", cmd.String()))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", out, err)
	}
	b.log.Debug(fmt.Sprintf("Command took %s", time.Since(startTime)))
	return nil
}
