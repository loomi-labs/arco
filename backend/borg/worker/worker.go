package worker

import (
	"arco/backend/borg/types"
	"fmt"
	"go.uber.org/zap"
)

type Worker struct {
	binaryPath   string
	log          *zap.SugaredLogger
	channels     *types.Channels
	shutdownChan chan struct{}
}

func NewWorker(log *zap.SugaredLogger, channels *types.Channels) (*Worker, *types.Channels) {
	return &Worker{
		binaryPath:   "bin/borg-linuxnewer64",
		log:          log,
		channels:     channels,
		shutdownChan: make(chan struct{}),
	}, channels
}

func (d *Worker) Run() {
	d.log.Info("Starting worker")

	for {
		select {
		case job := <-d.channels.StartBackup:
			d.log.Info("Starting backup job")
			go runBackup(job, d.channels.FinishBackup)
		case result := <-d.channels.FinishBackup:
			duration := result.EndTime.Sub(result.StartTime)
			if result.Err != nil {
				d.log.Error(fmt.Sprintf("Backup job failed after %s: %s", duration, result.Err))
			} else {
				d.log.Info(fmt.Sprintf("Backup job completed in %s", duration))
			}
			d.log.Debug(fmt.Sprintf("Command: %s", result.Cmd))
			d.channels.Notification <- fmt.Sprintf("Backup job completed in %s", duration)
		case <-d.shutdownChan:
			d.log.Debug("Shutting down background tasks")
			return
		}
	}
}

func (d *Worker) Stop() {
	d.log.Info("Stopping worker")
	close(d.shutdownChan)
}
