package borg

import (
	"arco/backend/ent"
	"arco/backend/ent/backupprofile"
	"arco/backend/ent/repository"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func (b *BorgClient) NewBackupProfile() (*ent.BackupProfile, error) {
	hostname, _ := os.Hostname()
	return b.client.BackupProfile.Create().
		SetName(hostname).
		SetPrefix(hostname).
		SetDirectories([]string{}).
		SetHasPeriodicBackups(true).
		//SetPeriodicBackupTime(time.Date(0, 0, 0, 9, 0, 0, 0, time.Local)).
		Save(context.Background())
}

func (b *BorgClient) GetDirectorySuggestions() []string {
	home, _ := os.UserHomeDir()
	if home != "" {
		return []string{home}
	}
	return []string{}
}

func (b *BorgClient) GetBackupProfile(id int) (*ent.BackupProfile, error) {
	return b.client.BackupProfile.
		Query().
		WithRepositories().
		Where(backupprofile.ID(id)).Only(context.Background())
}

func (b *BorgClient) GetBackupProfiles() ([]*ent.BackupProfile, error) {
	return b.client.BackupProfile.Query().All(context.Background())
}

func (b *BorgClient) SaveBackupProfile(backup ent.BackupProfile) error {
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

type backupJob struct {
	backupProfileId int
	repositoryId    int
	repoUrl         string
	repoPassword    string
	hostname        string
	directories     []string
	binaryPath      string
}

type finishBackupJob struct {
	backupProfileId int
	repositoryId    int
	startTime       time.Time
	endTime         time.Time
	cmd             string
	err             error
}

func runBackup(backupJob backupJob, finishBackupChannel chan finishBackupJob) {
	result := finishBackupJob{
		backupProfileId: backupJob.backupProfileId,
		repositoryId:    backupJob.repositoryId,
		startTime:       time.Now(),
		err:             nil,
	}
	defer func() {
		result.endTime = time.Now()
		finishBackupChannel <- result
	}()

	name := fmt.Sprintf("%s-%s", backupJob.hostname, time.Now().In(time.Local).Format("2006-01-02-15-04-05"))

	cmd := exec.Command(backupJob.binaryPath, "create", "--stats", fmt.Sprintf("%s::%s", backupJob.repoUrl, name), strings.Join(backupJob.directories, " "))
	cmd.Env = createEnv(backupJob.repoPassword)
	result.cmd = cmd.String()

	out, err := cmd.CombinedOutput()
	if err != nil {
		result.err = fmt.Errorf("%s: %s", out, err)
	}
}

func (b *BorgClient) RunBackup(backupProfileId int, repositoryId int) error {
	repo, err := b.client.Repository.
		Query().
		Where(repository.And(
			repository.ID(repositoryId),
			repository.HasBackupprofilesWith(backupprofile.ID(backupProfileId)),
		)).
		WithBackupprofiles(func(q *ent.BackupProfileQuery) {
			q.Limit(1)
			q.Where(backupprofile.ID(backupProfileId))
		}).
		Only(context.Background())
	if err != nil {
		return err
	}
	if len(repo.Edges.Backupprofiles) != 1 {
		return fmt.Errorf("repository does not have the backup profile")
	}

	backupProfile := repo.Edges.Backupprofiles[0]
	if !backupProfile.IsSetupComplete {
		return fmt.Errorf("backup profile is not complete")
	}

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	b.startBackupChannel <- backupJob{
		backupProfileId: backupProfileId,
		repositoryId:    repositoryId,
		repoUrl:         repo.URL,
		repoPassword:    repo.Password,
		hostname:        hostname,
		directories:     backupProfile.Directories,
		binaryPath:      b.binaryPath,
	}
	return nil
}

func (b *BorgClient) RunBackups(backupProfileId int) error {
	backupProfile, err := b.GetBackupProfile(backupProfileId)
	if err != nil {
		return err
	}
	if !backupProfile.IsSetupComplete {
		return fmt.Errorf("backup profile is not setup")
	}

	for _, repo := range backupProfile.Edges.Repositories {
		err := b.RunBackup(backupProfileId, repo.ID)
		if err != nil {
			return err
		}
	}
	return nil
}
