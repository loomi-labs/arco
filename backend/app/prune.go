package app

import (
	"arco/backend/types"
	"arco/backend/util"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func (b *BackupClient) runPruneJob(pruneJob types.PruneJob) {
	repoLock := b.state.GetRepoLock(pruneJob.Id)
	repoLock.Lock()
	defer repoLock.Unlock()
	defer b.state.DeleteRepoLock(pruneJob.Id)
	b.state.AddRunningPruneJob(b.ctx, pruneJob.Id)
	defer b.state.RemoveRunningBackup(pruneJob.Id)

	// Prepare prune command
	cmd := exec.CommandContext(b.ctx, pruneJob.BinaryPath, "prune", "--list", "--keep-daily", "3", "--keep-weekly", "4", "--glob-archives", fmt.Sprintf("'%s-*'", pruneJob.Prefix), pruneJob.RepoUrl)
	cmd.Env = util.BorgEnv{}.WithPassword(pruneJob.RepoPassword).AsList()

	// Run prune command
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err)).Error()
		b.state.AddNotification(errMsg, LevelError)

		// There is no need to continue with the compact job if the prune job failed
		return
	} else {
		b.log.LogCmdEnd(cmd.String(), startTime)
		b.state.AddNotification(fmt.Sprintf("Prune job completed in %s", time.Since(startTime)), LevelInfo)
	}

	// Prepare compact command
	cmd = exec.CommandContext(b.ctx, pruneJob.BinaryPath, "compact", pruneJob.RepoUrl)
	cmd.Env = util.BorgEnv{}.WithPassword(pruneJob.RepoPassword).AsList()
	b.log.Debug("Command: ", cmd.String())

	// Run compact command
	startTime = b.log.LogCmdStart(cmd.String())
	out, err = cmd.CombinedOutput()
	if err != nil {
		errMsg := b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err)).Error()
		b.state.AddNotification(errMsg, LevelError)
	} else {
		b.log.LogCmdEnd(cmd.String(), startTime)
		b.state.AddNotification(fmt.Sprintf("Compact job completed in %s", time.Since(startTime)), LevelInfo)
	}
}

func (b *BackupClient) PruneBackup(backupProfileId int, repositoryId int) error {
	repo, err := b.getRepoWithCompletedBackupProfile(repositoryId, backupProfileId)
	if err != nil {
		return err
	}
	backupProfile := repo.Edges.BackupProfiles[0]

	bId := types.BackupIdentifier{
		BackupProfileId: backupProfileId,
		RepositoryId:    repositoryId,
	}
	if canRun, reason := b.state.CanRunPruneJob(bId); !canRun {
		return fmt.Errorf(reason)
	}

	go b.runPruneJob(types.PruneJob{
		Id:           bId,
		RepoUrl:      repo.URL,
		RepoPassword: repo.Password,
		Prefix:       backupProfile.Prefix,
		BinaryPath:   b.config.BorgPath,
	})
	return nil
}

func (b *BackupClient) PruneBackups(backupProfileId int) error {
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

func (b *BackupClient) DryRunPruneBackup(backupProfileId int, repositoryId int) ([]PruneInfo, error) {
	return []PruneInfo{}, fmt.Errorf("not implemented")

	repo, err := b.getRepoWithCompletedBackupProfile(repositoryId, backupProfileId)
	if err != nil {
		return []PruneInfo{}, err
	}
	backupProfile := repo.Edges.BackupProfiles[0]

	// Prepare prune command (dry-run)
	cmd := exec.CommandContext(b.ctx, b.config.BorgPath, "prune", "-v", "--dry-run", "--list", "--keep-daily=1", "--keep-weekly=1", fmt.Sprintf("--glob-archives='%s-*'", backupProfile.Prefix), repo.URL)
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

func (b *BackupClient) DryRunPruneBackups(backupProfileId int) ([]PruneInfo, error) {
	return []PruneInfo{}, fmt.Errorf("not implemented")

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
