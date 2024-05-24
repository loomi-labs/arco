package worker

import (
	"arco/backend/borg/types"
	"arco/backend/borg/util"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func runBackup(backupJob types.BackupJob, finishBackupChannel chan types.FinishBackupJob) {
	result := types.FinishBackupJob{
		BackupProfileId: backupJob.BackupProfileId,
		RepositoryId:    backupJob.RepositoryId,
		StartTime:       time.Now(),
		Err:             nil,
	}
	defer func() {
		result.EndTime = time.Now()
		finishBackupChannel <- result
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
