package client

import (
	"arco/backend/borg/types"
	"arco/backend/borg/util"
	"arco/backend/ent"
	"arco/backend/ent/backupprofile"
	"arco/backend/ent/repository"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
)

func (b *BorgClient) NewBackupProfile() (*ent.BackupProfile, error) {
	hostname, _ := os.Hostname()
	return b.db.BackupProfile.Create().
		SetName(hostname).
		SetPrefix(hostname).
		SetDirectories([]string{}).
		SetHasPeriodicBackups(true).
		//SetPeriodicBackupTime(time.Date(0, 0, 0, 9, 0, 0, 0, time.Local)).
		Save(b.ctx)
}

func (b *BorgClient) GetDirectorySuggestions() []string {
	home, _ := os.UserHomeDir()
	if home != "" {
		return []string{home}
	}
	return []string{}
}

func (b *BorgClient) GetBackupProfile(id int) (*ent.BackupProfile, error) {
	return b.db.BackupProfile.
		Query().
		WithRepositories().
		Where(backupprofile.ID(id)).Only(b.ctx)
}

func (b *BorgClient) GetBackupProfiles() ([]*ent.BackupProfile, error) {
	return b.db.BackupProfile.Query().All(b.ctx)
}

func (b *BorgClient) SaveBackupProfile(backup ent.BackupProfile) error {
	_, err := b.db.BackupProfile.
		UpdateOneID(backup.ID).
		SetName(backup.Name).
		SetPrefix(backup.Prefix).
		SetDirectories(backup.Directories).
		SetHasPeriodicBackups(backup.HasPeriodicBackups).
		//SetPeriodicBackupTime(backup.PeriodicBackupTime).
		SetIsSetupComplete(backup.IsSetupComplete).
		Save(b.ctx)
	return err
}

func (b *BorgClient) getRepoWithCompletedBackupProfile(repoId int, backupProfileId int) (*ent.Repository, error) {
	repo, err := b.db.Repository.
		Query().
		Where(repository.And(
			repository.ID(repoId),
			repository.HasBackupprofilesWith(backupprofile.ID(backupProfileId)),
		)).
		WithBackupprofiles(func(q *ent.BackupProfileQuery) {
			q.Limit(1)
			q.Where(backupprofile.ID(backupProfileId))
		}).
		Only(b.ctx)
	if err != nil {
		return nil, err
	}
	if len(repo.Edges.Backupprofiles) != 1 {
		return nil, fmt.Errorf("repository does not have the backup profile")
	}
	if !repo.Edges.Backupprofiles[0].IsSetupComplete {
		return nil, fmt.Errorf("backup profile is not complete")
	}
	return repo, nil
}

func (b *BorgClient) RunBackup(backupProfileId int, repositoryId int) error {
	repo, err := b.getRepoWithCompletedBackupProfile(repositoryId, backupProfileId)
	if err != nil {
		return err
	}
	backupProfile := repo.Edges.Backupprofiles[0]

	bId := types.BackupIdentifier{
		BackupProfileId: backupProfileId,
		RepositoryId:    repositoryId,
	}
	if slices.Contains(b.runningBackups, bId) {
		return fmt.Errorf("backup is already running")
	}
	if slices.Contains(b.occupiedRepos, repositoryId) {
		return fmt.Errorf("repository is busy")
	}

	b.runningBackups = append(b.runningBackups, bId)
	b.occupiedRepos = append(b.occupiedRepos, repositoryId)

	b.inChan.StartBackup <- types.BackupJob{
		Id:           bId,
		RepoUrl:      repo.URL,
		RepoPassword: repo.Password,
		Prefix:       backupProfile.Prefix,
		Directories:  backupProfile.Directories,
		BinaryPath:   b.binaryPath,
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

func (b *BorgClient) PruneBackup(backupProfileId int, repositoryId int) error {
	repo, err := b.getRepoWithCompletedBackupProfile(repositoryId, backupProfileId)
	if err != nil {
		return err
	}
	backupProfile := repo.Edges.Backupprofiles[0]

	bId := types.BackupIdentifier{
		BackupProfileId: backupProfileId,
		RepositoryId:    repositoryId,
	}
	if slices.Contains(b.runningPruneJobs, bId) {
		return fmt.Errorf("prune job is already running")
	}
	if slices.Contains(b.occupiedRepos, repositoryId) {
		return fmt.Errorf("repository is busy")
	}

	b.runningPruneJobs = append(b.runningPruneJobs, bId)
	b.occupiedRepos = append(b.occupiedRepos, repositoryId)

	b.inChan.StartPrune <- types.PruneJob{
		Id:           bId,
		RepoUrl:      repo.URL,
		RepoPassword: repo.Password,
		Prefix:       backupProfile.Prefix,
		BinaryPath:   b.binaryPath,
	}
	return nil
}

func (b *BorgClient) PruneBackups(backupProfileId int) error {
	backupProfile, err := b.GetBackupProfile(backupProfileId)
	if err != nil {
		return err
	}
	if !backupProfile.IsSetupComplete {
		return fmt.Errorf("backup profile is not setup")
	}

	for _, repo := range backupProfile.Edges.Repositories {
		err := b.PruneBackup(backupProfileId, repo.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

type PruneInfo struct {
	Name   string
	Pruned bool
	Reason string
}

func parsePruneOutput(output string) []PruneInfo {
	// TODO: parsing of the output is not working correctly. There is no json output... for now let's just not use pruning info at all
	lines := strings.Split(output, "\n")
	var pruneInfos []PruneInfo

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Split the line into fields using at least five spaces as the separator
		fields := strings.SplitN(line, "     ", 3)
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
		if len(fields) != 3 {
			fmt.Println("Error parsing line:", line)
			continue
		}

		pruneInfo := PruneInfo{
			Name:   fields[1],
			Pruned: strings.HasPrefix(fields[0], "Would prune"),
		}

		// If not pruned, get the reason
		if !pruneInfo.Pruned {
			pruneInfo.Reason = fields[0]
		}

		pruneInfos = append(pruneInfos, pruneInfo)
	}

	return pruneInfos
}

func (b *BorgClient) DryRunPruneBackup(backupProfileId int, repositoryId int) ([]PruneInfo, error) {
	repo, err := b.getRepoWithCompletedBackupProfile(repositoryId, backupProfileId)
	if err != nil {
		return []PruneInfo{}, err
	}
	backupProfile := repo.Edges.Backupprofiles[0]

	// Prepare prune command (dry-run)
	cmd := exec.CommandContext(b.ctx, b.binaryPath, "prune", "-v", "--dry-run", "--list", "--keep-daily=1", "--keep-weekly=1", fmt.Sprintf("--glob-archives='%s-*'", backupProfile.Prefix), repo.URL)
	cmd.Env = util.BorgEnv{}.WithPassword(repo.Password).AsList()
	b.log.Debug("Command: ", cmd.String())
	// TODO: this is somehow not working when invoked with go (it works on the command line) -> fix this and parse the output

	// Run prune command (dry-run)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return []PruneInfo{}, fmt.Errorf("%s: %s", out, err)
	}
	return parsePruneOutput(string(out)), nil
}

func (b *BorgClient) DryRunPruneBackups(backupProfileId int) ([]PruneInfo, error) {
	var result []PruneInfo
	backupProfile, err := b.GetBackupProfile(backupProfileId)
	if err != nil {
		return result, err
	}
	if !backupProfile.IsSetupComplete {
		return result, fmt.Errorf("backup profile is not setup")
	}

	for _, repo := range backupProfile.Edges.Repositories {
		out, err := b.DryRunPruneBackup(backupProfileId, repo.ID)
		if err != nil {
			return result, err
		}
		result = append(result, out...)
	}
	return result, nil
}
