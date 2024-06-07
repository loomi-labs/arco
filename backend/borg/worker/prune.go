package worker

import (
	"arco/backend/borg/types"
	"arco/backend/borg/util"
	"context"
	"fmt"
	"os/exec"
	"time"
)

func (d *Worker) runPrune(ctx context.Context, pruneJob types.PruneJob) {
	result := types.FinishPruneJob{
		Id:        pruneJob.Id,
		StartTime: time.Now(),
		PruneErr:  nil,
	}
	defer func() {
		result.EndTime = time.Now()
		d.outChan.FinishPrune <- result
	}()

	// Prepare prune command
	cmd := exec.CommandContext(ctx, pruneJob.BinaryPath, "prune", "--list", "--keep-daily", "3", "--keep-weekly", "4", "--glob-archives", fmt.Sprintf("'%s-*'", pruneJob.Prefix), pruneJob.RepoUrl)
	cmd.Env = util.BorgEnv{}.WithPassword(pruneJob.RepoPassword).AsList()
	result.PruneCmd = cmd.String()

	// Run prune command
	startTime := d.log.LogCmdStart(result.PruneCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		result.PruneErr = d.log.LogCmdError(result.PruneCmd, startTime, fmt.Errorf("%s: %s", out, err))
	}
	d.log.LogCmdEnd(result.PruneCmd, startTime)

	// Prepare compact command
	cmd = exec.CommandContext(ctx, pruneJob.BinaryPath, "compact", pruneJob.RepoUrl)
	cmd.Env = util.BorgEnv{}.WithPassword(pruneJob.RepoPassword).AsList()
	result.CompactCmd = cmd.String()
	d.log.Debug("Command: ", result.CompactCmd)

	// Run compact command
	startTime = d.log.LogCmdStart(result.CompactCmd)
	out, err = cmd.CombinedOutput()
	if err != nil {
		result.CompactErr = d.log.LogCmdError(result.CompactCmd, startTime, fmt.Errorf("%s: %s", out, err))
	}
	d.log.LogCmdEnd(result.CompactCmd, startTime)
}
