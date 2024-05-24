package worker

import (
	"arco/backend/borg/types"
	"arco/backend/borg/util"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func (d *Worker) runBackup(ctx context.Context, backupJob types.BackupJob) {
	d.log.Info("Starting backup job")
	result := types.FinishBackupJob{
		Id:        backupJob.Id,
		StartTime: time.Now(),
		Err:       nil,
	}
	defer func() {
		result.EndTime = time.Now()
		d.outChan.FinishBackup <- result
	}()

	name := fmt.Sprintf("%s-%s", backupJob.Hostname, time.Now().In(time.Local).Format("2006-01-02-15-04-05"))

	cmd := exec.CommandContext(ctx, backupJob.BinaryPath, "create", fmt.Sprintf("%s::%s", backupJob.RepoUrl, name), strings.Join(backupJob.Directories, " "))
	cmd.Env = util.CreateEnv(backupJob.RepoPassword)
	result.Cmd = cmd.String()
	d.log.Debug("Command: ", result.Cmd)

	out, err := cmd.CombinedOutput()
	if err != nil {
		result.Err = fmt.Errorf("%s: %s", out, err)
	}
}
