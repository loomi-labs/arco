package daemon

import (
	"arco/backend/borg/types"
	"fmt"
	"go.uber.org/zap"
)

type Daemon struct {
	binaryPath string
	log        *zap.SugaredLogger
	channels   *types.Channels
}

func NewDaemon(log *zap.SugaredLogger) (*Daemon, *types.Channels) {
	channels := &types.Channels{
		ShutdownChannel:     make(chan struct{}),
		StartBackupChannel:  make(chan types.BackupJob),
		FinishBackupChannel: make(chan types.FinishBackupJob),
		NotificationChannel: make(chan string),
	}
	return &Daemon{
		binaryPath: "bin/borg-linuxnewer64",
		log:        log,
		channels:   channels,
	}, channels
}

func (d *Daemon) StartDaemon() {
	d.log.Info("Starting Borg daemon")

	for {
		select {
		case job := <-d.channels.StartBackupChannel:
			d.log.Info("Starting backup job")
			go runBackup(job, d.channels.FinishBackupChannel)
		case result := <-d.channels.FinishBackupChannel:
			duration := result.EndTime.Sub(result.StartTime)
			if result.Err != nil {
				d.log.Error(fmt.Sprintf("Backup job failed after %s: %s", duration, result.Err))
			} else {
				d.log.Info(fmt.Sprintf("Backup job completed in %s", duration))
			}
			d.log.Debug(fmt.Sprintf("Command: %s", result.Cmd))
			d.channels.NotificationChannel <- fmt.Sprintf("Backup job completed in %s", duration)
		case <-d.channels.ShutdownChannel:
			d.log.Debug("Shutting down background tasks")
			return
		}
	}
}

func (d *Daemon) StopDaemon() {
	d.log.Info("Stopping Borg daemon")
	close(d.channels.ShutdownChannel)
}
