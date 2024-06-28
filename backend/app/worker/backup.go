package worker

import (
	"arco/backend/app/types"
	"arco/backend/app/util"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func (d *Worker) runBackup(ctx context.Context, backupJob types.BackupJob) {
	result := types.FinishBackupJob{
		Id:        backupJob.Id,
		StartTime: time.Now(),
		Err:       nil,
	}
	defer func() {
		result.EndTime = time.Now()
		d.outChan.FinishBackup <- result
	}()

	// Prepare backup command
	name := fmt.Sprintf("%s-%s", backupJob.Prefix, time.Now().In(time.Local).Format("2006-01-02-15-04-05"))
	cmd := exec.CommandContext(ctx, backupJob.BinaryPath, "create", fmt.Sprintf("%s::%s", backupJob.RepoUrl, name), strings.Join(backupJob.Directories, " "))
	cmd.Env = util.BorgEnv{}.WithPassword(backupJob.RepoPassword).AsList()
	result.Cmd = cmd.String()

	// Run backup command
	startTime := d.log.LogCmdStart(result.Cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		result.Err = d.log.LogCmdError(result.Cmd, startTime, fmt.Errorf("%s: %s", out, err))
	}
	d.log.LogCmdEnd(result.Cmd, startTime)
}
