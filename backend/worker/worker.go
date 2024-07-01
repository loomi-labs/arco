package worker

import (
	"arco/backend/types"
	"arco/backend/util"
	"context"
	"go.uber.org/zap"
)

type Worker struct {
	log          *util.CmdLogger
	borgPath     string
	actionChans  *types.ActionChannels
	resultChans  *types.ResultChannels
	shutdownChan chan struct{}
}

func NewWorker(log *zap.SugaredLogger, borgPath string, inChan *types.ActionChannels, outChan *types.ResultChannels) *Worker {
	return &Worker{
		log:          util.NewCmdLogger(log),
		borgPath:     borgPath,
		actionChans:  inChan,
		resultChans:  outChan,
		shutdownChan: make(chan struct{}),
	}
}

func (d *Worker) Run() {
	d.log.Info("Starting worker")
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	for {
		select {
		case job := <-d.actionChans.StartBackup:
			go d.runBackup(ctx, job)
		case job := <-d.actionChans.StartPrune:
			go d.runPrune(ctx, job)
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
