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
	inChan       *types.InputChannels
	outChan      *types.OutputChannels
	shutdownChan chan struct{}
}

func NewWorker(log *zap.SugaredLogger, borgPath string, inChan *types.InputChannels, outChan *types.OutputChannels) *Worker {
	return &Worker{
		log:          util.NewCmdLogger(log),
		borgPath:     borgPath,
		inChan:       inChan,
		outChan:      outChan,
		shutdownChan: make(chan struct{}),
	}
}

func (d *Worker) Run() {
	d.log.Info("Starting worker")
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	for {
		select {
		case job := <-d.inChan.StartBackup:
			go d.runBackup(ctx, job)
		case job := <-d.inChan.StartPrune:
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
