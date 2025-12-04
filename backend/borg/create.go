package borg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	gocmd "github.com/go-cmd/cmd"
	"github.com/loomi-labs/arco/backend/borg/types"
	"github.com/loomi-labs/arco/backend/ent/backupprofile"
)

// Create creates a new backup in the repository.
// It is long running and should be run in a goroutine.
func (b *borg) Create(ctx context.Context, repository, password, prefix string, backupPaths, excludePaths []string, excludeCaches bool, compressionMode backupprofile.CompressionMode, compressionLevel *int, ch chan types.BackupProgress) (string, *types.Status) {
	archivePath := fmt.Sprintf("%s::%s%s", repository, prefix, time.Now().In(time.Local).Format("2006-01-02-15-04-05"))

	// Count the total files to backup
	totalFiles, borgStatus, err := b.countBackupFiles(ctx, archivePath, password, backupPaths, excludePaths, excludeCaches, compressionMode, compressionLevel)
	if err != nil {
		return "", newStatusWithError(err)
	}
	if !borgStatus.IsCompletedWithSuccess() {
		return "", borgStatus
	}

	// Prepare backup command
	cmdStr := []string{
		"create",     // https://borgbackup.readthedocs.io/en/stable/usage/create.html#borg-create
		"--progress", // Outputs continuous progress messages
		"--log-json", // Outputs JSON log messages
	}

	// Add compression flag if enabled
	if compressionFlag := buildCompressionFlag(compressionMode, compressionLevel); compressionFlag != "" {
		cmdStr = append(cmdStr, compressionFlag)
	}

	// Add archive path and backup paths
	cmdStr = append(cmdStr, archivePath)
	cmdStr = append(cmdStr, backupPaths...)

	// Add exclude paths
	for _, excludeDir := range excludePaths {
		cmdStr = append(cmdStr, "--exclude", excludeDir) // Paths and files that will be ignored
	}

	// Add exclude caches flag
	if excludeCaches {
		cmdStr = append(cmdStr, "--exclude-caches")
	}

	options := gocmd.Options{Buffered: false, Streaming: true}
	cmd := gocmd.NewCmdOptions(options, b.path, cmdStr...)
	cmdLog := fmt.Sprintf("%s %s", b.path, strings.Join(cmdStr, " "))
	cmd.Env = NewEnv(b.sshPrivateKeys).WithPassword(password).AsList()

	// Run backup command
	b.log.LogCmdStart(cmdLog)
	statusChan := cmd.Start()

	go decodeBackupProgress(cmd, totalFiles, ch)

	select {
	case <-ctx.Done():
		// If the context gets cancelled we stop the command
		err = cmd.Stop()
		if err != nil {
			b.log.Errorf("error stopping command: %v", err)
		}

		// We still have to wait for the command to finish
		_ = <-statusChan

		// We don't care about the real status of the borg operation because we canceled it
		borgStatus = newStatusWithCanceled()
		return archivePath, b.log.LogCmdStatus(ctx, borgStatus, cmdLog, time.Duration(cmd.Status().Runtime))
	case _ = <-statusChan:
		// Break in case the command completes
		break
	}

	// If we are here the command has completed
	status := cmd.Status()
	borgStatus = gocmdToStatus(status, "")
	return archivePath, b.log.LogCmdStatus(ctx, borgStatus, cmdLog, time.Duration(status.Runtime))
}

// decodeBackupProgress decodes the progress messages from borg and sends them to the channel.
func decodeBackupProgress(cmd *gocmd.Cmd, totalFiles int, ch chan<- types.BackupProgress) {
	defer close(ch)
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
func (b *borg) countBackupFiles(ctx context.Context, archiveName, password string, backupPaths, excludePaths []string, excludeCaches bool, compressionMode backupprofile.CompressionMode, compressionLevel *int) (int, *types.Status, error) {
	cmdStr := []string{
		"create",     // https://borgbackup.readthedocs.io/en/stable/usage/create.html#borg-create
		"--dry-run",  // Simulate the backup
		"--list",     // List the files and directories to be backed up
		"--log-json", // Outputs JSON log messages
	}

	// Add compression flag if enabled
	if compressionFlag := buildCompressionFlag(compressionMode, compressionLevel); compressionFlag != "" {
		cmdStr = append(cmdStr, compressionFlag)
	}

	// Add archive name and backup paths
	cmdStr = append(cmdStr, archiveName)
	cmdStr = append(cmdStr, backupPaths...)

	// Add exclude paths
	for _, excludeDir := range excludePaths {
		cmdStr = append(cmdStr, "--exclude", excludeDir) // Paths and files that will be ignored
	}

	// Add exclude caches flag
	if excludeCaches {
		cmdStr = append(cmdStr, "--exclude-caches")
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
	result := gocmdToStatus(status, "")
	if result.HasError() || result.HasBeenCanceled {
		return 0, b.log.LogCmdStatus(ctx, result, cmdLog, time.Duration(status.Runtime)), nil
	}

	b.log.LogCmdStatus(ctx, result, cmdLog, time.Duration(status.Runtime))

	select {
	case totalFiles := <-fileCountChan:
		return totalFiles, result, nil
	case <-time.After(10 * time.Second):
		return 0, result, fmt.Errorf("timeout reached while counting files")
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

// buildCompressionFlag builds the --compression flag for borg based on mode and level.
// Returns empty string if compression is disabled (none or empty mode).
func buildCompressionFlag(mode backupprofile.CompressionMode, level *int) string {
	// No compression
	if mode == backupprofile.CompressionModeNone || mode == "" {
		return ""
	}

	// lz4 doesn't support levels
	if mode == backupprofile.CompressionModeLz4 {
		return "--compression=lz4"
	}

	// Algorithms with levels (zstd, zlib, lzma)
	if level != nil {
		return fmt.Sprintf("--compression=%s,%d", string(mode), *level)
	}

	// Default levels if not specified
	switch mode {
	case backupprofile.CompressionModeZstd:
		return "--compression=zstd,3"
	case backupprofile.CompressionModeZlib:
		return "--compression=zlib,6"
	case backupprofile.CompressionModeLzma:
		return "--compression=lzma,6"
	case backupprofile.CompressionModeNone:
		return ""
	case backupprofile.CompressionModeLz4:
		return "--compression=lz4"
	}

	// Fallback: just use the algorithm name
	return fmt.Sprintf("--compression=%s", string(mode))
}
