package worker

import (
	"arco/backend/borg/types"
	"arco/backend/borg/util"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func (d *Worker) runBackup(backupJob types.BackupJob) {
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

	cmd := exec.Command(backupJob.BinaryPath, "create", "--stats", fmt.Sprintf("%s::%s", backupJob.RepoUrl, name), strings.Join(backupJob.Directories, " "))
	cmd.Env = util.CreateEnv(backupJob.RepoPassword)
	result.Cmd = cmd.String()

	out, err := cmd.CombinedOutput()
	if err != nil {
		result.Err = fmt.Errorf("%s: %s", out, err)
	}
}
