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
	"time"
)

// Create creates a new backup in the repository.
// It is long running and should be run in a goroutine.
func (b *Borg) Create(ctx context.Context, repoUrl, password, prefix string, directories []string, ch chan BackupProgress) error {
	// Count the total files to backup
	totalFiles, err := b.countBackupFiles(ctx, repoUrl, password, prefix, directories)
	if err != nil {
		return err
	}

	// Prepare backup command
	name := fmt.Sprintf("%s-%s", prefix, time.Now().In(time.Local).Format("2006-01-02-15-04-05"))
	cmd := exec.CommandContext(ctx, b.path, append([]string{
		"create",     // https://borgbackup.readthedocs.io/en/stable/usage/create.html#borg-create
		"--progress", // Outputs continuous progress messages
		"--log-json", // Outputs JSON log messages
		fmt.Sprintf("%s::%s", repoUrl, name)},
		directories...,
	)...)
	cmd.Env = Env{}.WithPassword(password).AsList()

	// Run backup command
	startTime := b.log.LogCmdStart(cmd.String())

	// Borg streams JSON messages to stderr
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return b.log.LogCmdError(cmd.String(), startTime, err)
	}

	err = cmd.Start()
	if err != nil {
		return b.log.LogCmdError(cmd.String(), startTime, err)
	}

	scanner := bufio.NewScanner(stderr)
	scanner.Split(bufio.ScanLines)
	progressDecoder(scanner, totalFiles, ch)

	err = cmd.Wait()
	if err != nil {
		return b.log.LogCmdError(cmd.String(), startTime, err)
	}
	b.log.LogCmdEnd(cmd.String(), startTime)
	return nil
}

type BackupProgress struct {
	TotalFiles     int `json:"totalFiles"`
	ProcessedFiles int `json:"processedFiles"`
}

// progressDecoder decodes the progress messages from Borg and sends them to the channel.
func progressDecoder(scanner *bufio.Scanner, totalFiles int, ch chan<- BackupProgress) {
	defer close(ch)
	for scanner.Scan() {
		data := scanner.Text()

		var typeMsg Type
		decoder := json.NewDecoder(strings.NewReader(data))
		err := decoder.Decode(&typeMsg)
		if err != nil {
			// Continue if we can't decode the JSON
			continue
		}
		if typeMsg.Type != "archive_progress" {
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
func (b *Borg) countBackupFiles(ctx context.Context, repoUrl, password, prefix string, directories []string) (int, error) {
	name := fmt.Sprintf("%s-%s", prefix, time.Now().In(time.Local).Format("2006-01-02-15-04-05"))
	cmd := exec.CommandContext(ctx, b.path, append([]string{
		"create",     // https://borgbackup.readthedocs.io/en/stable/usage/create.html#borg-create
		"--dry-run",  // Simulate the backup
		"--list",     // List the files and directories to be backed up
		"--log-json", // Outputs JSON log messages
		fmt.Sprintf("%s::%s", repoUrl, name)},
		directories...,
	)...)
	cmd.Env = Env{}.WithPassword(password).AsList()

	// Run backup command
	startTime := b.log.LogCmdStart(cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, b.log.LogCmdError(cmd.String(), startTime, fmt.Errorf("%s: %s", out, err))
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
