package borg

import (
	"context"
	"encoding/json"
	"fmt"
	gocmd "github.com/go-cmd/cmd"
	"github.com/loomi-labs/arco/backend/borg/types"
	"os"
	"strings"
	"time"
)

// Create creates a new backup in the repository.
// It is long running and should be run in a goroutine.
func (b *borg) Create(ctx context.Context, repository, password, prefix string, backupPaths, excludePaths []string, ch chan types.BackupProgress) (string, error) {
	archiveName := fmt.Sprintf("%s::%s%s", repository, prefix, time.Now().In(time.Local).Format("2006-01-02-15-04-05"))

	// Count the total files to backup
	totalFiles, err := b.countBackupFiles(ctx, archiveName, password, backupPaths, excludePaths)
	if err != nil {
		return "", err
	}

	// Prepare backup command
	cmdStr := append([]string{
		"create",     // https://borgbackup.readthedocs.io/en/stable/usage/create.html#borg-create
		"--progress", // Outputs continuous progress messages
		"--log-json", // Outputs JSON log messages
		archiveName,
	}, backupPaths...,
	)
	for _, excludeDir := range excludePaths {
		cmdStr = append(cmdStr, "--exclude", excludeDir) // Paths and files that will be ignored
	}

	options := gocmd.Options{Buffered: false, Streaming: true}
	cmd := gocmd.NewCmdOptions(options, b.path, cmdStr...)
	cmdLog := fmt.Sprintf("%s %s", b.path, strings.Join(cmdStr, " "))
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	// Run backup command
	b.log.LogCmdStart(cmdLog)
	statusChan := cmd.Start()

	go decodeBackupProgress(cmd, totalFiles, ch, b.log)

	select {
	case <-ctx.Done():
		// If the context gets cancelled we stop the command
		err = cmd.Stop()
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
	status := cmd.Status()
	if status.Error != nil {
		return archiveName, b.log.LogCmdErrorD(ctx, cmdLog, time.Duration(status.Runtime), status.Error)
	}
	if !status.Complete {
		b.log.LogCmdCancelledD(cmdLog, time.Duration(status.Runtime))
		return archiveName, CancelErr{}
	}
	b.log.LogCmdEndD(cmdLog, time.Duration(status.Runtime))
	return archiveName, nil
}

// decodeBackupProgress decodes the progress messages from borg and sends them to the channel.
func decodeBackupProgress(cmd *gocmd.Cmd, totalFiles int, ch chan<- types.BackupProgress, log *CmdLogger) {
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
			if types.JSONType(typeMsg.Type) != types.ArchiveProgressType {
				// We only care about archive progress
				continue
			}

			var archiveProgress types.ArchiveProgress
			if err := json.Unmarshal([]byte(data), &archiveProgress); err != nil {
				// Skip errors
				continue
			}
			if archiveProgress.Finished {
				ch <- types.BackupProgress{TotalFiles: totalFiles, ProcessedFiles: totalFiles}
			} else if totalFiles > 0 && archiveProgress.NFiles > 0 {
				ch <- types.BackupProgress{TotalFiles: totalFiles, ProcessedFiles: archiveProgress.NFiles}
			}
		case <-cmd.Done():
			return
		}
	}
}

// countBackupFiles counts the number of files that will be backed up.
// We use the --dry-run flag to simulate the backup and count the files.
func (b *borg) countBackupFiles(ctx context.Context, archiveName, password string, backupPaths, excludePaths []string) (int, error) {
	cmdStr := append([]string{
		"create",     // https://borgbackup.readthedocs.io/en/stable/usage/create.html#borg-create
		"--dry-run",  // Simulate the backup
		"--list",     // List the files and directories to be backed up
		"--log-json", // Outputs JSON log messages
		archiveName},
		backupPaths...,
	)
	for _, excludeDir := range excludePaths {
		cmdStr = append(cmdStr, "--exclude", excludeDir) // Paths and files that will be ignored
	}

	options := gocmd.Options{Buffered: false, Streaming: true}
	cmd := gocmd.NewCmdOptions(options, b.path, cmdStr...)
	cmdLog := fmt.Sprintf("%s %s", b.path, strings.Join(cmdStr, " "))
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	// Run dry-run command
	b.log.LogCmdStart(cmdLog)
	statusChan := cmd.Start()
	fileCountChan := make(chan int)

	go countFiles(cmd, fileCountChan)

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
	status := cmd.Status()
	if status.Error != nil {
		return 0, b.log.LogCmdErrorD(ctx, cmdLog, time.Duration(status.Runtime), status.Error)
	}
	if !status.Complete {
		b.log.LogCmdCancelledD(cmdLog, time.Duration(status.Runtime))
		return 0, CancelErr{}
	}
	b.log.LogCmdEndD(cmdLog, time.Duration(status.Runtime))

	select {
	case totalFiles := <-fileCountChan:
		return totalFiles, nil
	case <-time.After(10 * time.Second):
		return 0, fmt.Errorf("timeout reached while counting files")
	}
}

// countFiles counts the number of files in the output of the borg --list command.
func countFiles(cmd *gocmd.Cmd, fileCountChan chan int) {
	totalFiles := 0

	for {
		select {
		case _ = <-cmd.Stdout:
			// ignore stdout (info comes through stderr)
		case data := <-cmd.Stderr:
			var fileStatus types.FileStatus
			if err := json.Unmarshal([]byte(data), &fileStatus); err != nil {
				// Skip errors
				continue
			}

			stat, err := os.Stat(fileStatus.Path)
			if err != nil || stat.IsDir() {
				// Skip errors and directories
				continue
			}
			totalFiles++
		case <-cmd.Done():
			fileCountChan <- totalFiles
			return
		}
	}
}
