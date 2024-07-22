package borg

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

type PruneResult struct {
	PruneArchives []*PruneArchive
	KeepArchives  []*KeepArchive
}

type PruneArchive struct {
	Name string
}

type KeepArchive struct {
	Name   string
	Reason string
}

// TODO: get this from the config
var pruneOptions = []string{
	"--keep-daily",
	"3",
	"--keep-weekly",
	"4",
}

func (b *Borg) Prune(ctx context.Context, repoUrl, password, prefix string, ch chan PruneResult) error {
	defer close(ch)

	// Prepare prune command
	cmdStr := []string{
		"prune",           // https://borgbackup.readthedocs.io/en/stable/usage/prune.html#borg-prune
		"--list",          // List archives to be pruned
		"--log-json",      // Outputs JSON log messages
		"--glob-archives", // Match archives by glob pattern
		fmt.Sprintf("%s-*", prefix),
	}
	cmdStr = append(cmdStr, pruneOptions...)
	cmdStr = append(cmdStr, repoUrl)
	cmd := exec.CommandContext(ctx, b.path, cmdStr...)
	cmd.Env = Env{}.WithPassword(password).AsList()

	// Add cancel functionality
	hasBeenCanceled := false
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		hasBeenCanceled = true
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
	}

	// Run prune command
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		if hasBeenCanceled {
			b.log.LogCmdCancelled(cmd.String(), startTime)
			return CancelErr{}
		}
		return b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	} else {
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		b.log.LogCmdEnd(cmd.String(), startTime)

		ch <- decodePruneOutput(scanner)

		// Run compact to free up space
		return b.Compact(ctx, repoUrl, password)
	}
}

//type PruneArchive struct {
//	Name   string
//	Pruned bool
//	Reason string
//}
//
//func parsePruneOutput(output string) []PruneArchive {
//	// TODO: parsing of the output is not working correctly. There is no json output... for now let's just not use pruning info at all
//	lines := strings.Split(output, "\n")
//	var pruneInfos []PruneArchive
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
//		pruneInfo := PruneArchive{
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
//func (b *BackupClient) DryRunPruneBackup(backupProfileId int, repositoryId int) ([]PruneArchive, error) {
//	return []PruneArchive{}, fmt.Errorf("not implemented")
//
//	repo, err := b.getRepoWithCompletedBackupProfile(repositoryId, backupProfileId)
//	if err != nil {
//		return []PruneArchive{}, err
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
//		return []PruneArchive{}, fmt.Errorf("%s: %s", out, err)
//	}
//	return parsePruneOutput(string(out)), nil
//}
//
//func (b *BackupClient) DryRunPruneBackups(backupProfileId int) ([]PruneArchive, error) {
//	return []PruneArchive{}, fmt.Errorf("not implemented")
//
//	var result []PruneArchive
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

// decodeBackupProgress decodes the progress messages from Borg and sends them to the channel.
func decodePruneOutput(scanner *bufio.Scanner) PruneResult {
	var prunedArchives []*PruneArchive
	var keptArchives []*KeepArchive
	for scanner.Scan() {
		data := scanner.Text()

		var typeMsg Type
		decoder := json.NewDecoder(strings.NewReader(data))
		err := decoder.Decode(&typeMsg)
		if err != nil {
			// Continue if we can't decode the JSON
			continue
		}
		if JSONType(typeMsg.Type) != LogMessageType {
			// We only care about log messages
			continue
		}

		var LogMsg LogMessage
		decoder = json.NewDecoder(strings.NewReader(data))
		err = decoder.Decode(&LogMsg)
		if err != nil {
			continue
		}
		prune, keep := parsePruneOutput(LogMsg)
		if prune != nil {
			prunedArchives = append(prunedArchives, prune)
		}
		if keep != nil {
			keptArchives = append(keptArchives, keep)
		}
	}
	return PruneResult{
		PruneArchives: prunedArchives,
		KeepArchives:  keptArchives,
	}
}

func parsePruneOutput(logMsg LogMessage) (*PruneArchive, *KeepArchive) {
	if strings.HasPrefix(logMsg.Message, "Would prune") || strings.HasPrefix(logMsg.Message, "Pruning") {
		return &PruneArchive{
			Name: parsePruneName(logMsg.Message),
		}, nil
	} else if strings.HasPrefix(logMsg.Message, "Keeping") {
		return nil, &KeepArchive{
			Name:   parsePruneName(logMsg.Message),
			Reason: parsePruneReason(logMsg.Message),
		}
	} else {
		return nil, nil
	}
}

func parsePruneName(msg string) string {
	if len(msg) < 45 {
		return ""
	}
	return strings.Split(msg[45:], " ")[0]
}

func parsePruneReason(msg string) string {
	// Example: "Keeping archive (rule: daily #1):            archive-name-2024-07-22-19-43-49             Mon, 2024-07-22 19:43:49 [6daa8ab1e2898805f4f44e15996e4893fce21b57fcfd91436bd39f3d03b412ba]"
	// Return: "daily #1"

	// Find the "()" and return the content (without "rule: ")
	start := strings.Index(msg, "(rule: ")
	end := strings.Index(msg, ")")
	if start == -1 || end == -1 {
		return ""
	}
	return msg[start+7 : end]
}
