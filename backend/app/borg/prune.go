package borg

import (
	"context"
	"fmt"
	"os/exec"
)

func (b *Borg) Prune(ctx context.Context, repoUrl, password, prefix string) error {
	// Prepare prune command
	cmd := exec.CommandContext(ctx, b.path,
		"prune",
		"--list",
		"--keep-daily",
		"3",
		"--keep-weekly",
		"4",
		"--glob-archives",
		fmt.Sprintf("'%s-*'", prefix),
		repoUrl,
	)
	cmd.Env = Env{}.WithPassword(password).AsList()

	// Run prune command
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	} else {
		b.log.LogCmdEnd(cmd.String(), startTime)

		// Run compact to free up space
		return b.Compact(ctx, repoUrl, password)
	}
}

//type PruneInfo struct {
//	Name   string
//	Pruned bool
//	Reason string
//}
//
//func parsePruneOutput(output string) []PruneInfo {
//	// TODO: parsing of the output is not working correctly. There is no json output... for now let's just not use pruning info at all
//	lines := strings.Split(output, "\n")
//	var pruneInfos []PruneInfo
//
//	for _, line := range lines {
//		// Skip empty lines
//		if strings.TrimSpace(line) == "" {
//			continue
//		}
//
//		// Split the line into fields using at least five spaces as the separator
//		fields := strings.SplitN(line, "     ", 3)
//		for i := range fields {
//			fields[i] = strings.TrimSpace(fields[i])
//		}
//		if len(fields) != 3 {
//			fmt.Println("Error parsing line:", line)
//			continue
//		}
//
//		pruneInfo := PruneInfo{
//			Name:   fields[1],
//			Pruned: strings.HasPrefix(fields[0], "Would prune"),
//		}
//
//		// If not pruned, get the reason
//		if !pruneInfo.Pruned {
//			pruneInfo.Reason = fields[0]
//		}
//
//		pruneInfos = append(pruneInfos, pruneInfo)
//	}
//
//	return pruneInfos
//}
//
//func (b *BackupClient) DryRunPruneBackup(backupProfileId int, repositoryId int) ([]PruneInfo, error) {
//	return []PruneInfo{}, fmt.Errorf("not implemented")
//
//	repo, err := b.getRepoWithCompletedBackupProfile(repositoryId, backupProfileId)
//	if err != nil {
//		return []PruneInfo{}, err
//	}
//	backupProfile := repo.Edges.BackupProfiles[0]
//
//	// Prepare prune command (dry-run)
//	cmd := exec.CommandContext(b.ctx, b.config.BorgPath, "prune", "-v", "--dry-run", "--list", "--keep-daily=1", "--keep-weekly=1", fmt.Sprintf("--glob-archives='%s-*'", backupProfile.Prefix), repo.URL)
//	cmd.Env = util.Env{}.WithPassword(repo.Password).AsList()
//	b.log.Debug("Command: ", cmd.String())
//	// TODO: this is somehow not working when invoked with go (it works on the command line) -> fix this and parse the output
//
//	// Run prune command (dry-run)
//	out, err := cmd.CombinedOutput()
//	if err != nil {
//		return []PruneInfo{}, fmt.Errorf("%s: %s", out, err)
//	}
//	return parsePruneOutput(string(out)), nil
//}
//
//func (b *BackupClient) DryRunPruneBackups(backupProfileId int) ([]PruneInfo, error) {
//	return []PruneInfo{}, fmt.Errorf("not implemented")
//
//	var result []PruneInfo
//	backupProfile, err := b.GetBackupProfile(backupProfileId)
//	if err != nil {
//		return result, err
//	}
//	if !backupProfile.IsSetupComplete {
//		return result, fmt.Errorf("backup profile is not setup")
//	}
//
//	for _, repo := range backupProfile.Edges.Repositories {
//		out, err := b.DryRunPruneBackup(backupProfileId, repo.ID)
//		if err != nil {
//			return result, err
//		}
//		result = append(result, out...)
//	}
//	return result, nil
//}
