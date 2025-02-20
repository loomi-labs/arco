package borg

import (
	"context"
	"encoding/json"
	"fmt"
	gocmd "github.com/go-cmd/cmd"
	"github.com/loomi-labs/arco/backend/borg/types"
	"strings"
	"time"
)

func (b *borg) Prune(ctx context.Context, repository string, password string, prefix string, pruneOptions []string, isDryRun bool, ch chan types.PruneResult) error {
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

	options := gocmd.Options{Buffered: false, Streaming: true}
	cmd := gocmd.NewCmdOptions(options, b.path, cmdStr...)
	cmdLog := fmt.Sprintf("%s %s", b.path, strings.Join(cmdStr, " "))
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	// Run command
	statusChan := cmd.Start()

	go decodePruneOutput(cmd, isDryRun, ch)

	select {
	case <-ctx.Done():
		// If the context gets cancelled we stop the command
		err := cmd.Stop()
		if err != nil {
			b.log.Errorf("error stopping command: %v", err)
		}

		// We still have to wait for the command to finish
		_ = <-statusChan
	case _ = <-statusChan:
		// Break in case the command completes
		break
	}

	// If we are here the command has completed or the context has been cancelled
	b.log.LogCmdStart(cmdLog)
	status := cmd.Status()
	if status.Error != nil {
		return b.log.LogCmdErrorD(ctx, cmdLog, time.Duration(status.Runtime), status.Error)
	}
	if !status.Complete {
		b.log.LogCmdCancelledD(cmdLog, time.Duration(status.Runtime))
		return CancelErr{}
	}
	b.log.LogCmdEndD(cmdLog, time.Duration(status.Runtime))

	if isDryRun {
		return nil
	}

	// Run compact to free up space
	return b.Compact(ctx, repository, password)
}

// decodePruneOutput decodes the progress messages from borg and sends them to the channel.
func decodePruneOutput(cmd *gocmd.Cmd, isDryRun bool, ch chan types.PruneResult) {
	defer close(ch)

	var prunedArchives []*types.PruneArchive
	var keptArchives []*types.KeepArchive

	for {
		select {
		case _ = <-cmd.Stdout:
			// ignore stdout (info comes through stderr)
		case data := <-cmd.Stderr:
			var typeMsg types.Type
			if err := json.Unmarshal([]byte(data), &typeMsg); err != nil {
				// Skip errors
				continue
			}
			if types.JSONType(typeMsg.Type) != types.LogMessageType {
				// We only care about log messages
				continue
			}

			var logMsg types.LogMessage
			if err := json.Unmarshal([]byte(data), &logMsg); err != nil {
				// Skip errors
				continue
			}
			prune, keep := parsePruneOutput(logMsg)
			if prune != nil {
				prunedArchives = append(prunedArchives, prune)
			}
			if keep != nil {
				keptArchives = append(keptArchives, keep)
			}
		case <-cmd.Done():
			ch <- types.PruneResult{
				IsDryRun:      isDryRun,
				PruneArchives: prunedArchives,
				KeepArchives:  keptArchives,
			}
			return
		}
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
