package borg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"timebender/backend/ent"
	"timebender/backend/ent/backupprofile"
)

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
		b.log.Info(fmt.Sprintf("Running command: %s", cmd.String()))
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s: %s", out, err)
		}
		b.log.Info(fmt.Sprintf("Command took %s", time.Since(startTime)))
	}
	return nil
}
