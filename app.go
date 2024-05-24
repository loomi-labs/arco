package main

import (
	"arco/backend/borg"
	"context"
)

// App struct
type App struct {
	ctx        context.Context
	BorgClient *borg.Client
}

// NewApp creates a new App application struct
func NewApp(borg *borg.Client) *App {
	return &App{
		BorgClient: borg,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.BorgClient.StartDaemon()
}

// shutdown is called when the app is shutting down
func (a *App) shutdown(ctx context.Context) {
	a.BorgClient.StopDaemon()
}
