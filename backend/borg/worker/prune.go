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
	d.log.Info("Starting prune job")
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
	d.log.Debug("Command: ", result.PruneCmd)

	// Run prune command
	out, err := cmd.CombinedOutput()
	if err != nil {
		result.PruneErr = fmt.Errorf("%s: %s", out, err)
		d.log.Error("Error running prune command: ", result.PruneErr)
	}
	d.log.Debug("Prune job finished")

	// Prepare compact command
	cmd = exec.CommandContext(ctx, pruneJob.BinaryPath, "compact", pruneJob.RepoUrl)
	cmd.Env = util.BorgEnv{}.WithPassword(pruneJob.RepoPassword).AsList()
	result.CompactCmd = cmd.String()
	d.log.Debug("Command: ", result.CompactCmd)

	// Run compact command
	out, err = cmd.CombinedOutput()
	if err != nil {
		result.CompactErr = fmt.Errorf("%s: %s", out, err)
		d.log.Error("Error running compact command: ", result.CompactErr)
	}
	d.log.Debug("Compact job finished")
}
