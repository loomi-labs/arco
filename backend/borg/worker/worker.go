package worker

import (
	"arco/backend/borg/types"
	"go.uber.org/zap"
)

type Worker struct {
	binaryPath   string
	log          *zap.SugaredLogger
	inChan       *types.InputChannels
	outChan      *types.OutputChannels
	shutdownChan chan struct{}
}

func NewWorker(log *zap.SugaredLogger, inChan *types.InputChannels, outChan *types.OutputChannels) *Worker {
	return &Worker{
		binaryPath:   "bin/borg-linuxnewer64",
		log:          log,
		inChan:       inChan,
		outChan:      outChan,
		shutdownChan: make(chan struct{}),
	}
}

func (d *Worker) Run() {
	d.log.Info("Starting worker")

	for {
		select {
		case job := <-d.inChan.StartBackup:
			go d.runBackup(job)
		case <-d.shutdownChan:
			d.log.Debug("Shutting down worker")
			return
		}
	}
}

func (d *Worker) Stop() {
	d.log.Info("Stopping worker")
	close(d.shutdownChan)
}
