package borg

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type BackupProgress struct {
	TotalFiles     int `json:"totalFiles"`
	ProcessedFiles int `json:"processedFiles"`
}

// Create creates a new backup in the repository.
// It is long running and should be run in a goroutine.
func (b *borg) Create(ctx context.Context, repository, password, prefix string, backupPaths, excludePaths []string, ch chan BackupProgress) (string, error) {
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
	cmd := exec.CommandContext(ctx, b.path, cmdStr...)
	cmd.Env = Env{}.WithPassword(password).AsList()

	// Add cancel functionality
	hasBeenCanceled := false
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		hasBeenCanceled = true
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
	}

	// Run backup command
	startTime := b.log.LogCmdStart(cmd.String())

	// borg streams JSON messages to stderr
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return archiveName, b.log.LogCmdError(ctx, cmd.String(), startTime, err)
	}

	err = cmd.Start()
	if err != nil {
		return archiveName, b.log.LogCmdError(ctx, cmd.String(), startTime, err)
	}

	scanner := bufio.NewScanner(stderr)
	scanner.Split(bufio.ScanLines)
	decodeBackupProgress(scanner, totalFiles, ch)

	err = cmd.Wait()
	if err != nil {
		if hasBeenCanceled {
			b.log.LogCmdCancelled(cmd.String(), startTime)
			return archiveName, CancelErr{}
		}
		return archiveName, b.log.LogCmdError(ctx, cmd.String(), startTime, err)
	}

	b.log.LogCmdEnd(cmd.String(), startTime)
	return archiveName, nil
}

// decodeBackupProgress decodes the progress messages from borg and sends them to the channel.
func decodeBackupProgress(scanner *bufio.Scanner, totalFiles int, ch chan<- BackupProgress) {
	for scanner.Scan() {
		data := scanner.Text()

		var typeMsg Type
		decoder := json.NewDecoder(strings.NewReader(data))
		err := decoder.Decode(&typeMsg)
		if err != nil {
			// Continue if we can't decode the JSON
			continue
		}
		if JSONType(typeMsg.Type) != ArchiveProgressType {
			// We only care about archive progress
			continue
		}

		var archiveProgress ArchiveProgress
		decoder = json.NewDecoder(strings.NewReader(data))
		err = decoder.Decode(&archiveProgress)
		if err != nil {
			continue
		}
		if archiveProgress.Finished {
			ch <- BackupProgress{TotalFiles: totalFiles, ProcessedFiles: totalFiles}
		} else if totalFiles > 0 && archiveProgress.NFiles > 0 {
			ch <- BackupProgress{TotalFiles: totalFiles, ProcessedFiles: archiveProgress.NFiles}
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
	cmd := exec.CommandContext(ctx, b.path, cmdStr...)
	cmd.Env = Env{}.WithPassword(password).AsList()

	// Add cancel functionality
	hasBeenCanceled := false
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		hasBeenCanceled = true
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
	}

	// Run backup command
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		if hasBeenCanceled {
			b.log.LogCmdCancelled(cmd.String(), startTime)
			return 0, CancelErr{}
		}
		return 0, b.log.LogCmdError(ctx, cmd.String(), startTime, err)
	}
	b.log.LogCmdEnd(cmd.String(), startTime)

	// Count the files of the output
	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Split(bufio.ScanLines)
	return countFiles(scanner), nil
}

// countFiles counts the number of files in the output of the borg --list command.
func countFiles(scanner *bufio.Scanner) int {
	totalFiles := 0
	for scanner.Scan() {
		data := scanner.Text()

		var fileStatus FileStatus
		decoder := json.NewDecoder(strings.NewReader(data))
		err := decoder.Decode(&fileStatus)
		if err != nil {
			continue
		}

		stat, err := os.Stat(fileStatus.Path)
		if err != nil || stat.IsDir() {
			// Skip errors and directories
			continue
		}
		totalFiles++
	}
	return totalFiles
}
