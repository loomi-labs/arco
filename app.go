package main

import (
	"arco/backend/borg/client"
	"context"
)

// App struct
type App struct {
	ctx        context.Context
	BorgClient *client.BorgClient
}

// NewApp creates a new App application struct
func NewApp(borg *client.BorgClient) *App {
	return &App{
		BorgClient: borg,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// shutdown is called when the app is shutting down
func (a *App) shutdown(ctx context.Context) {
}
