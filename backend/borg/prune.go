package borg

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/loomi-labs/arco/backend/borg/types"
	"os/exec"
	"strings"
	"syscall"
)

func (b *borg) Prune(ctx context.Context, repository string, password string, prefix string, pruneOptions []string, isDryRun bool, ch chan types.PruneResult) error {
	defer close(ch)

	if len(pruneOptions) == 0 {
		return fmt.Errorf("pruneOptions must not be empty")
	}

	// Prepare prune command
	cmdStr := []string{
		"prune",           // https://borgbackup.readthedocs.io/en/stable/usage/prune.html#borg-prune
		"--list",          // List archives to be pruned
		"--log-json",      // Outputs JSON log messages
		"--glob-archives", // Match archives by glob pattern
		fmt.Sprintf("%s*", prefix),
	}

	if isDryRun {
		cmdStr = append(cmdStr, "--dry-run")
	}

	cmdStr = append(cmdStr, pruneOptions...)
	cmdStr = append(cmdStr, repository)
	cmd := exec.CommandContext(ctx, b.path, cmdStr...)
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

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
		return b.log.LogCmdError(ctx, cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
	} else {
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		b.log.LogCmdEnd(cmd.String(), startTime)

		ch <- decodePruneOutput(scanner, isDryRun)

		if isDryRun {
			return nil
		}

		// Run compact to free up space
		return b.Compact(ctx, repository, password)
	}
}

// decodeBackupProgress decodes the progress messages from borg and sends them to the channel.
func decodePruneOutput(scanner *bufio.Scanner, isDryRun bool) types.PruneResult {
	var prunedArchives []*types.PruneArchive
	var keptArchives []*types.KeepArchive
	for scanner.Scan() {
		data := scanner.Text()

		var typeMsg types.Type
		decoder := json.NewDecoder(strings.NewReader(data))
		err := decoder.Decode(&typeMsg)
		if err != nil {
			// Continue if we can't decode the JSON
			continue
		}
		if types.JSONType(typeMsg.Type) != types.LogMessageType {
			// We only care about log messages
			continue
		}

		var LogMsg types.LogMessage
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
	return types.PruneResult{
		IsDryRun:      isDryRun,
		PruneArchives: prunedArchives,
		KeepArchives:  keptArchives,
	}
}

func parsePruneOutput(logMsg types.LogMessage) (*types.PruneArchive, *types.KeepArchive) {
	if strings.HasPrefix(logMsg.Message, "Would prune") || strings.HasPrefix(logMsg.Message, "Pruning") {
		return &types.PruneArchive{
			Name: parsePruneName(logMsg.Message),
		}, nil
	} else if strings.HasPrefix(logMsg.Message, "Keeping") {
		return nil, &types.KeepArchive{
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
