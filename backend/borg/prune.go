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
	IsDryRun      bool
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

func (b *borg) Prune(ctx context.Context, repoUrl, password, prefix string, isDryRun bool, ch chan PruneResult) error {
	defer close(ch)

	// Prepare prune command
	cmdStr := []string{
		"prune",           // https://borgbackup.readthedocs.io/en/stable/usage/prune.html#borg-prune
		"--list",          // List archives to be pruned
		"--log-json",      // Outputs JSON log messages
		"--glob-archives", // Match archives by glob pattern
		fmt.Sprintf("%s-*", prefix),
	}

	if isDryRun {
		cmdStr = append(cmdStr, "--dry-run")
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

		ch <- decodePruneOutput(scanner, isDryRun)

		if isDryRun {
			return nil
		}

		// Run compact to free up space
		return b.Compact(ctx, repoUrl, password)
	}
}

// decodeBackupProgress decodes the progress messages from borg and sends them to the channel.
func decodePruneOutput(scanner *bufio.Scanner, isDryRun bool) PruneResult {
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
		IsDryRun:      isDryRun,
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
