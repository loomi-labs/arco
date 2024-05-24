package main

import (
	"arco/backend/borg/client"
	"arco/backend/borg/worker"
	"context"
)

// App struct
type App struct {
	ctx        context.Context
	BorgClient *client.BorgClient
	borgWoker  *worker.Worker
}

// NewApp creates a new App application struct
func NewApp(borg *client.BorgClient, worker *worker.Worker) *App {
	return &App{
		BorgClient: borg,
		borgWoker:  worker,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.BorgClient.Startup(ctx)
}

// shutdown is called when the app is shutting down
func (a *App) shutdown(_ context.Context) {
	a.borgWoker.Stop()
}
